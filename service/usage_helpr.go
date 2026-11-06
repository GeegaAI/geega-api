package service

import (
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
)

//func GetPromptTokens(textRequest dto.GeneralOpenAIRequest, relayMode int) (int, error) {
//	switch relayMode {
//	case constant.RelayModeChatCompletions:
//		return CountTokenMessages(textRequest.Messages, textRequest.Model)
//	case constant.RelayModeCompletions:
//		return CountTokenInput(textRequest.Prompt, textRequest.Model), nil
//	case constant.RelayModeModerations:
//		return CountTokenInput(textRequest.Input, textRequest.Model), nil
//	}
//	return 0, errors.New("unknown relay mode")
//}

// ResponseText2Usage 使用本地估算构造整份 usage。
// 注意：调用方若只缺失输入侧或输出侧，应改用 MarkPromptTokensLocal /
// MarkCompletionTokensLocal 精确标记，避免把上游已返回的那一侧也标成本地估算。
func ResponseText2Usage(c *gin.Context, responseText string, modeName string, promptTokens int) *dto.Usage {
	return ResponseText2UsageWithReason(c, responseText, modeName, promptTokens, UsageFallbackUsageMissing)
}

// ResponseText2UsageWithReason 同 ResponseText2Usage，但允许调用方给出更具体的
// 退化原因（如 stream_usage_missing），便于日志聚合与告警。
func ResponseText2UsageWithReason(c *gin.Context, responseText string, modeName string, promptTokens int, reason string) *dto.Usage {
	MarkAllTokensLocal(c, reason)
	usage := &dto.Usage{}
	usage.PromptTokens = promptTokens
	usage.CompletionTokens = EstimateTokenByModel(modeName, responseText)
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}

func ValidUsage(usage *dto.Usage) bool {
	return usage != nil && (usage.PromptTokens != 0 || usage.CompletionTokens != 0)
}
