package service

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"
)

// usage fallback 原因码。写入日志 admin_info.usage_fallback，
// 作为「上游 usage 缺失 → 本地估算」的可观测指标；同时维护进程内计数器。
const (
	UsageFallbackUsageMissing       = "usage_missing"        // 上游完全未返回 usage
	UsageFallbackStreamUsageMissing = "stream_usage_missing" // 流式响应未返回 usage（常见于未开启 include_usage）
	UsageFallbackPromptMissing      = "prompt_missing"       // 上游只返回了输出侧 token
	UsageFallbackCompletionMissing  = "completion_missing"   // 上游只返回了输入侧 token
)

// UsageLocality 描述一次请求最终 usage 中，输入/输出各自是否来自本地估算。
// 历史版本只有一个 ContextKeyLocalCountTokens 总开关；这里拆成两个维度，
// 使「输入来自上游、输出为估算」的请求仍能保留精确的上游输入 token。
type UsageLocality struct {
	PromptLocal     bool
	CompletionLocal bool
}

// AnyLocal 表示至少有一侧不是上游原始值。
func (l UsageLocality) AnyLocal() bool {
	return l.PromptLocal || l.CompletionLocal
}

// AllLocal 表示输入与输出都来自本地估算。
func (l UsageLocality) AllLocal() bool {
	return l.PromptLocal && l.CompletionLocal
}

// UsageLocalityFromContext 读取当前请求的估算标记。
// 兼容旧路径：只设置了 ContextKeyLocalCountTokens 时，保守视为两侧均为本地估算。
func UsageLocalityFromContext(c *gin.Context) UsageLocality {
	if c == nil {
		return UsageLocality{}
	}
	locality := UsageLocality{
		PromptLocal:     common.GetContextKeyBool(c, constant.ContextKeyLocalPromptTokens),
		CompletionLocal: common.GetContextKeyBool(c, constant.ContextKeyLocalCompletionTokens),
	}
	if !locality.AnyLocal() && common.GetContextKeyBool(c, constant.ContextKeyLocalCountTokens) {
		return UsageLocality{PromptLocal: true, CompletionLocal: true}
	}
	return locality
}

// MarkPromptTokensLocal 将输入侧标记为本地估算，并记录原因。
func MarkPromptTokensLocal(c *gin.Context, reason string) {
	if c == nil {
		return
	}
	common.SetContextKey(c, constant.ContextKeyLocalPromptTokens, true)
	// 保留历史总开关，兼容只认识 local_count_tokens 的既有日志/前端逻辑。
	common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
	recordUsageFallback(c, reason)
}

// MarkCompletionTokensLocal 将输出侧标记为本地估算，并记录原因。
func MarkCompletionTokensLocal(c *gin.Context, reason string) {
	if c == nil {
		return
	}
	common.SetContextKey(c, constant.ContextKeyLocalCompletionTokens, true)
	common.SetContextKey(c, constant.ContextKeyLocalCountTokens, true)
	recordUsageFallback(c, reason)
}

// MarkAllTokensLocal 将输入与输出都标记为本地估算。
func MarkAllTokensLocal(c *gin.Context, reason string) {
	MarkPromptTokensLocal(c, reason)
	MarkCompletionTokensLocal(c, "")
}

// UsageFallbackReasonFromContext 返回本次请求累计的估算原因（逗号分隔）。
func UsageFallbackReasonFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return common.GetContextKeyString(c, constant.ContextKeyUsageFallbackReason)
}

// recordUsageFallback 记录原因码并累加进程内计数，同一请求同一原因只计一次。
func recordUsageFallback(c *gin.Context, reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}

	existing := UsageFallbackReasonFromContext(c)
	for _, item := range strings.Split(existing, ",") {
		if item == reason {
			return
		}
	}
	if existing == "" {
		common.SetContextKey(c, constant.ContextKeyUsageFallbackReason, reason)
	} else {
		common.SetContextKey(c, constant.ContextKeyUsageFallbackReason, existing+","+reason)
	}
	usageFallbackCounter(reason).Add(1)
	if isUsageFallbackWarning(reason) {
		logger.LogWarn(c, fmt.Sprintf("usage fallback to local estimate: reason=%s", reason))
	} else {
		logger.LogInfo(c, fmt.Sprintf("usage fallback to local estimate: reason=%s", reason))
	}
}

// isUsageFallbackWarning 区分严重程度：输入侧或整份 usage 缺失会影响计费准确性，
// 值得 warn；仅输出侧估算（客户端提前断开等）降为 info，避免日志噪声。
func isUsageFallbackWarning(reason string) bool {
	switch reason {
	case UsageFallbackPromptMissing, UsageFallbackUsageMissing, UsageFallbackStreamUsageMissing:
		return true
	default:
		return false
	}
}

// ResetUsageLocality 清空当前请求的估算标记与原因。
// 重试（换渠道）时必须调用：上一次失败尝试留下的 local 标记不能污染
// 本次成功尝试的结算与对账字段。
func ResetUsageLocality(c *gin.Context) {
	if c == nil {
		return
	}
	common.SetContextKey(c, constant.ContextKeyLocalCountTokens, false)
	common.SetContextKey(c, constant.ContextKeyLocalPromptTokens, false)
	common.SetContextKey(c, constant.ContextKeyLocalCompletionTokens, false)
	common.SetContextKey(c, constant.ContextKeyUsageFallbackReason, "")
}

var usageFallbackStats sync.Map // reason -> *atomic.Int64

func usageFallbackCounter(reason string) *atomic.Int64 {
	if counter, ok := usageFallbackStats.Load(reason); ok {
		return counter.(*atomic.Int64)
	}
	counter, _ := usageFallbackStats.LoadOrStore(reason, &atomic.Int64{})
	return counter.(*atomic.Int64)
}

// GetUsageFallbackStats 返回进程启动以来各估算原因的累计次数。
// 这是实时告警用的内存指标；持久化的按原因统计请使用日志
// other.admin_info.usage_fallback 或对账接口。
func GetUsageFallbackStats() map[string]int64 {
	stats := make(map[string]int64)
	usageFallbackStats.Range(func(key, value any) bool {
		stats[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})
	return stats
}

// ResetUsageFallbackStats 仅用于测试，清空进程内计数器。
func ResetUsageFallbackStats() {
	usageFallbackStats.Range(func(key, _ any) bool {
		usageFallbackStats.Delete(key)
		return true
	})
}
