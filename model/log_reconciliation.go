package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// UsageReconciliationGroup 对账聚合的原始分组，字段为 DB 侧 sum 值（未做口径换算）。
// 最细分组维度：数据来源 × 语义 × 渠道（供应商）× 模型。
type UsageReconciliationGroup struct {
	UsageBillingPath             string `json:"usage_billing_path"`
	UsageSemantic                string `json:"usage_semantic"`
	ChannelId                    int64  `json:"channel_id"`
	ChannelName                  string `json:"channel_name"` // 查询后从主库 channels 表回填
	ModelName                    string `json:"model_name"`
	RequestCount                 int64  `json:"request_count"`
	Quota                        int64  `json:"quota"`
	PromptTokens                 int64  `json:"prompt_tokens"`
	CompletionTokens             int64  `json:"completion_tokens"`
	CacheTokens                  int64  `json:"cache_tokens"`
	CacheCreationEffectiveTokens int64  `json:"cache_creation_effective_tokens"`
}

// UsageReconciliationRow 在原始分组上完成口径换算后的对账行。
// 对账核心是输入侧：上游计费口径输入 vs 系统记录输入。anthropic 语义 prompt_tokens
// 不含缓存，需加回 cache_read/cache_creation；openai/gemini 语义 prompt_tokens 即计费输入全量。
// 系统输出本身就是上游 usage 的照抄值，没有独立的上游输出口径，故不提供上游输出对比。
// 仅 Reconcilable 行的 upstream 字段有真实意义；不可对账行（语义未标注/估算/本地）
// upstream 恒为 0，避免默认"上游=系统"制造虚假对平。
type UsageReconciliationRow struct {
	UsageBillingPath    string `json:"usage_billing_path"`
	Semantic            string `json:"semantic"`
	ChannelId           int64  `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	ModelName           string `json:"model_name"`
	Estimated           bool   `json:"estimated"`
	Local               bool   `json:"local"`
	Reconcilable        bool   `json:"reconcilable"`
	RequestCount        int64  `json:"request_count"`
	Quota               int64  `json:"quota"`
	SystemInputTokens   int64  `json:"system_input_tokens"`
	SystemOutputTokens  int64  `json:"system_output_tokens"`
	UpstreamInputTokens int64  `json:"upstream_input_tokens"`
	// 仅 total 行填充：可对账请求的请求数
	ReconciledRequestCount int64 `json:"reconciled_request_count"`
}

func GetUsageReconciliationGroups(startTimestamp int64, endTimestamp int64) ([]UsageReconciliationGroup, error) {
	selectExpr := reconciliationPathExpr() + " AS usage_billing_path, " +
		reconciliationSemanticExpr() + " AS usage_semantic, " +
		"channel_id, model_name, " +
		"count(*) AS request_count, " +
		"COALESCE(sum(quota), 0) AS quota, " +
		"COALESCE(sum(prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(sum(completion_tokens), 0) AS completion_tokens, " +
		"COALESCE(sum(" + reconciliationOtherIntExpr("cache_tokens") + "), 0) AS cache_tokens, " +
		"COALESCE(sum(" + reconciliationEffectiveCacheCreationExpr() + "), 0) AS cache_creation_effective_tokens"
	tx := LOG_DB.Table("logs").
		Select(selectExpr).
		Where("type = ?", LogTypeConsume)
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	var groups []UsageReconciliationGroup
	err := tx.Group("usage_billing_path, usage_semantic, channel_id, model_name").Scan(&groups).Error
	if err != nil {
		return nil, err
	}
	// logs 表的 channel_name 为只读虚拟字段（渠道名在主库 channels 表），按既有模式批量回填
	channelIds := make(map[int]struct{})
	for i := range groups {
		if groups[i].ChannelId != 0 {
			channelIds[int(groups[i].ChannelId)] = struct{}{}
		}
	}
	if len(channelIds) > 0 {
		ids := make([]int, 0, len(channelIds))
		for id := range channelIds {
			ids = append(ids, id)
		}
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := DB.Table("channels").Select("id, name").Where("id IN ?", ids).Find(&channels).Error; err != nil {
			return nil, err
		}
		channelMap := make(map[int]string, len(channels))
		for _, channel := range channels {
			channelMap[channel.Id] = channel.Name
		}
		for i := range groups {
			groups[i].ChannelName = channelMap[int(groups[i].ChannelId)]
		}
	}
	return groups, nil
}

func DeriveUsageReconciliationRow(group UsageReconciliationGroup) UsageReconciliationRow {
	path := group.UsageBillingPath
	semantic := "unknown"
	switch {
	case strings.Contains(path, "anthropic") || group.UsageSemantic == "anthropic":
		semantic = "anthropic"
	case strings.Contains(path, "gemini") || group.UsageSemantic == "gemini":
		semantic = "gemini"
	case strings.Contains(path, "openai") || group.UsageSemantic == "openai":
		semantic = "openai"
	}
	estimated := strings.Contains(path, "estimated")
	local := path == "local"
	reconcilable := semantic != "unknown" && !estimated && !local
	row := UsageReconciliationRow{
		UsageBillingPath:   path,
		Semantic:           semantic,
		ChannelId:          group.ChannelId,
		ChannelName:        group.ChannelName,
		ModelName:          group.ModelName,
		Estimated:          estimated,
		Local:              local,
		Reconcilable:       reconcilable,
		RequestCount:       group.RequestCount,
		Quota:              group.Quota,
		SystemInputTokens:  group.PromptTokens,
		SystemOutputTokens: group.CompletionTokens,
	}
	if reconcilable {
		row.UpstreamInputTokens = group.PromptTokens
		if semantic == "anthropic" {
			row.UpstreamInputTokens = group.PromptTokens + group.CacheTokens + group.CacheCreationEffectiveTokens
		}
	}
	return row
}

func reconciliationPathExpr() string {
	switch common.LogDatabaseType() {
	case common.DatabaseTypeClickHouse:
		return "if(JSONExtractString(other, 'admin_info', 'usage_billing_path') = '', 'legacy', JSONExtractString(other, 'admin_info', 'usage_billing_path'))"
	case common.DatabaseTypePostgreSQL:
		return "COALESCE(NULLIF(other, '')::jsonb -> 'admin_info' ->> 'usage_billing_path', 'legacy')"
	case common.DatabaseTypeMySQL:
		return "COALESCE(IF(JSON_VALID(other), JSON_UNQUOTE(JSON_EXTRACT(other, '$.admin_info.usage_billing_path')), NULL), 'legacy')"
	default:
		return "COALESCE(CASE WHEN json_valid(other) THEN json_extract(other, '$.admin_info.usage_billing_path') END, 'legacy')"
	}
}

func reconciliationSemanticExpr() string {
	switch common.LogDatabaseType() {
	case common.DatabaseTypeClickHouse:
		return "JSONExtractString(other, 'usage_semantic')"
	case common.DatabaseTypePostgreSQL:
		return "NULLIF(other, '')::jsonb ->> 'usage_semantic'"
	case common.DatabaseTypeMySQL:
		return "IF(JSON_VALID(other), JSON_UNQUOTE(JSON_EXTRACT(other, '$.usage_semantic')), NULL)"
	default:
		return "CASE WHEN json_valid(other) THEN json_extract(other, '$.usage_semantic') END"
	}
}

// key 为编译期常量，无注入面
func reconciliationOtherIntExpr(key string) string {
	switch common.LogDatabaseType() {
	case common.DatabaseTypeClickHouse:
		return "JSONExtractInt(other, '" + key + "')"
	case common.DatabaseTypePostgreSQL:
		return "COALESCE((NULLIF(other, '')::jsonb ->> '" + key + "')::bigint, 0)"
	case common.DatabaseTypeMySQL:
		return "IF(JSON_VALID(other), IFNULL(JSON_UNQUOTE(JSON_EXTRACT(other, '$." + key + "')), 0), 0)"
	default:
		return "CASE WHEN json_valid(other) THEN COALESCE(json_extract(other, '$." + key + "'), 0) ELSE 0 END"
	}
}

// Anthropic 同时上报 cache_creation 总量与 5m/1h 拆分（拆分之和=总量），
// 逐行二取一防止重复累计；语义对齐 relaykit ClaudeUsage.GetCacheCreationTotalTokens
func reconciliationEffectiveCacheCreationExpr() string {
	total := reconciliationOtherIntExpr("cache_creation_tokens")
	split := reconciliationOtherIntExpr("cache_creation_tokens_5m") + " + " + reconciliationOtherIntExpr("cache_creation_tokens_1h")
	return "CASE WHEN " + total + " > 0 THEN " + total + " ELSE " + split + " END"
}
