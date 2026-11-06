package service

import (
	"strings"

	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/gin-gonic/gin"
)

const (
	usageBillingPathLocal              = "local"
	usageBillingPathUpstream           = "upstream"
	usageBillingPathOpenAI             = "billing-usage-openai"
	usageBillingPathOpenAIEstimated    = "billing-usage-openai-estimated"
	usageBillingPathAnthropic          = "billing-usage-anthropic"
	usageBillingPathAnthropicEstimated = "billing-usage-anthropic-estimated"
	usageBillingPathGemini             = "billing-usage-gemini"
	usageBillingPathGeminiEstimated    = "billing-usage-gemini-estimated"
)

func effectiveBillingUsage(usage *dto.Usage) *dto.Usage {
	if billingUsage, ok := usageFromBillingUsage(usage); ok {
		return billingUsage
	}
	return usage
}

// semantic 为该次请求的最终 usage 语义（来自上游标注或 relay 格式推断）。
// 上游真实返回 usage 但未装配 BillingUsage 时，按语义落 billing-usage-* 路径，
// 使对账页可覆盖 openai 兼容渠道；usage 为 nil（本地估算兜底）时保持 upstream 不可对账。
//
// 只有输入侧是本地估算时才落 local：对账比对的是输入，输入不可信就完全不可对账。
// 输出侧单独估算（输入来自上游）时落 *-estimated，并对账页通过 estimated 标记排除；
// 具体退化方向由 admin_info.local_prompt_tokens / local_completion_tokens 标注。
func usageBillingPathForLog(locality UsageLocality, semantic string, usage *dto.Usage) string {
	if locality.PromptLocal {
		return usageBillingPathLocal
	}
	effectiveUsage, ok := usageFromBillingUsage(usage)
	if !ok {
		// 没有原生快照时，输出侧估算只能靠 semantic 推断出 *-estimated，
		// 否则会误标成完全可对账的上游语义。
		if locality.CompletionLocal {
			switch semantic {
			case dto.BillingUsageSemanticAnthropic:
				return usageBillingPathAnthropicEstimated
			case dto.BillingUsageSemanticGemini:
				return usageBillingPathGeminiEstimated
			case dto.BillingUsageSemanticOpenAI:
				return usageBillingPathOpenAIEstimated
			}
			return usageBillingPathLocal
		}
		if usage != nil {
			switch semantic {
			case dto.BillingUsageSemanticAnthropic:
				return usageBillingPathAnthropic
			case dto.BillingUsageSemanticGemini:
				return usageBillingPathGemini
			case dto.BillingUsageSemanticOpenAI:
				return usageBillingPathOpenAI
			}
		}
		return usageBillingPathUpstream
	}

	switch effectiveUsage.UsageSemantic {
	case dto.BillingUsageSemanticOpenAI:
		if usage.BillingUsage.Estimated || locality.CompletionLocal {
			return usageBillingPathOpenAIEstimated
		}
		return usageBillingPathOpenAI
	case dto.BillingUsageSemanticAnthropic:
		if usage.BillingUsage.Estimated || locality.CompletionLocal {
			return usageBillingPathAnthropicEstimated
		}
		return usageBillingPathAnthropic
	case dto.BillingUsageSemanticGemini:
		if usage.BillingUsage.Estimated || locality.CompletionLocal {
			return usageBillingPathGeminiEstimated
		}
		return usageBillingPathGemini
	}

	return usageBillingPathUpstream
}

func appendUsageBillingPathForLog(ctx *gin.Context, other map[string]interface{}, locality UsageLocality, semantic string, usage *dto.Usage) {
	if other == nil {
		return
	}
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	if !ok || adminInfo == nil {
		adminInfo = make(map[string]interface{})
		other["admin_info"] = adminInfo
	}
	adminInfo["usage_billing_path"] = usageBillingPathForLog(locality, semantic, usage)
	if locality.PromptLocal {
		adminInfo["local_prompt_tokens"] = true
	}
	if locality.CompletionLocal {
		adminInfo["local_completion_tokens"] = true
	}
	if reason := UsageFallbackReasonFromContext(ctx); reason != "" {
		adminInfo["usage_fallback"] = reason
	}
}

func usageFromBillingUsage(usage *dto.Usage) (*dto.Usage, bool) {
	if usage == nil || usage.BillingUsage == nil {
		return nil, false
	}
	billingUsage := usage.BillingUsage
	source := strings.TrimSpace(billingUsage.Source)
	semantic := strings.TrimSpace(billingUsage.Semantic)

	if billingUsage.OpenAIUsage != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceOAIChat) ||
			strings.EqualFold(source, dto.BillingUsageSourceOAIResponses) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticOpenAI)) {
		return usageFromOpenAIBillingUsage(billingUsage), true
	}

	if billingUsage.ClaudeUsage != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceClaudeMessages) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticAnthropic)) {
		return usageFromClaudeBillingUsage(billingUsage), true
	}

	if billingUsage.GeminiUsageMetadata != nil &&
		(strings.EqualFold(source, dto.BillingUsageSourceGeminiChat) ||
			strings.EqualFold(semantic, dto.BillingUsageSemanticGemini)) {
		return usageFromGeminiBillingUsage(billingUsage), true
	}

	return nil, false
}

func usageFromOpenAIBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	usage := *billingUsage.OpenAIUsage
	if usage.PromptTokens == 0 && usage.InputTokens > 0 {
		usage.PromptTokens = usage.InputTokens
	}
	if usage.CompletionTokens == 0 && usage.OutputTokens > 0 {
		usage.CompletionTokens = usage.OutputTokens
	}
	if usage.InputTokens == 0 && usage.PromptTokens > 0 {
		usage.InputTokens = usage.PromptTokens
	}
	if usage.OutputTokens == 0 && usage.CompletionTokens > 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if inputDetails := usage.InputTokensDetails; inputDetails != nil {
		if usage.PromptTokensDetails.CachedTokens == 0 && inputDetails.CachedTokens > 0 {
			usage.PromptTokensDetails.CachedTokens = inputDetails.CachedTokens
		}
		if usage.PromptTokensDetails.CachedCreationTokens == 0 && inputDetails.CachedCreationTokens > 0 {
			usage.PromptTokensDetails.CachedCreationTokens = inputDetails.CachedCreationTokens
		}
		if usage.PromptTokensDetails.CacheWriteTokens == 0 && inputDetails.CacheWriteTokens > 0 {
			usage.PromptTokensDetails.CacheWriteTokens = inputDetails.CacheWriteTokens
		}
		if usage.PromptTokensDetails.TextTokens == 0 && inputDetails.TextTokens > 0 {
			usage.PromptTokensDetails.TextTokens = inputDetails.TextTokens
		}
		if usage.PromptTokensDetails.ImageTokens == 0 && inputDetails.ImageTokens > 0 {
			usage.PromptTokensDetails.ImageTokens = inputDetails.ImageTokens
		}
		if usage.PromptTokensDetails.AudioTokens == 0 && inputDetails.AudioTokens > 0 {
			usage.PromptTokensDetails.AudioTokens = inputDetails.AudioTokens
		}
	}
	if usage.PromptTokensDetails.CachedTokens == 0 && usage.PromptCacheHitTokens > 0 {
		usage.PromptTokensDetails.CachedTokens = usage.PromptCacheHitTokens
	}
	usage.UsageSemantic = dto.BillingUsageSemanticOpenAI
	usage.UsageSource = billingUsage.Source
	usage.BillingUsage = dto.CloneBillingUsage(billingUsage)
	return &usage
}

func usageFromClaudeBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	claudeUsage := billingUsage.ClaudeUsage
	cacheCreation5m := claudeUsage.GetCacheCreation5mTokens()
	if cacheCreation5m == 0 {
		cacheCreation5m = claudeUsage.ClaudeCacheCreation5mTokens
	}
	cacheCreation1h := claudeUsage.GetCacheCreation1hTokens()
	if cacheCreation1h == 0 {
		cacheCreation1h = claudeUsage.ClaudeCacheCreation1hTokens
	}

	usage := &dto.Usage{
		PromptTokens:                claudeUsage.InputTokens,
		CompletionTokens:            claudeUsage.OutputTokens,
		TotalTokens:                 claudeUsage.InputTokens + claudeUsage.OutputTokens,
		InputTokens:                 claudeUsage.InputTokens + claudeUsage.CacheReadInputTokens + claudeUsage.CacheCreationInputTokens,
		OutputTokens:                claudeUsage.OutputTokens,
		UsageSemantic:               dto.BillingUsageSemanticAnthropic,
		UsageSource:                 dto.BillingUsageSourceClaudeMessages,
		BillingUsage:                dto.CloneBillingUsage(billingUsage),
		ClaudeCacheCreation5mTokens: cacheCreation5m,
		ClaudeCacheCreation1hTokens: cacheCreation1h,
	}
	usage.PromptTokensDetails.CachedTokens = claudeUsage.CacheReadInputTokens
	usage.PromptTokensDetails.CachedCreationTokens = claudeUsage.CacheCreationInputTokens
	return usage
}

func usageFromGeminiBillingUsage(billingUsage *dto.BillingUsage) *dto.Usage {
	metadata := *billingUsage.GeminiUsageMetadata
	promptTokens := metadata.PromptTokenCount + metadata.ToolUsePromptTokenCount
	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: metadata.CandidatesTokenCount + metadata.ThoughtsTokenCount,
		TotalTokens:      metadata.TotalTokenCount,
		UsageSemantic:    dto.BillingUsageSemanticGemini,
		UsageSource:      dto.BillingUsageSourceGeminiChat,
		BillingUsage:     dto.CloneBillingUsage(billingUsage),
	}
	usage.CompletionTokenDetails.ReasoningTokens = metadata.ThoughtsTokenCount
	usage.PromptTokensDetails.CachedTokens = metadata.CachedContentTokenCount

	for _, detail := range metadata.PromptTokensDetails {
		addGeminiInputTokenDetail(&usage.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.ToolUsePromptTokensDetails {
		addGeminiInputTokenDetail(&usage.PromptTokensDetails, detail)
	}
	for _, detail := range metadata.CandidatesTokensDetails {
		switch detail.Modality {
		case "IMAGE":
			usage.CompletionTokenDetails.ImageTokens += detail.TokenCount
		case "AUDIO":
			usage.CompletionTokenDetails.AudioTokens += detail.TokenCount
		case "TEXT":
			usage.CompletionTokenDetails.TextTokens += detail.TokenCount
		}
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	} else if usage.CompletionTokens <= 0 {
		usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
	}
	if usage.PromptTokens > 0 && usage.PromptTokensDetails.TextTokens == 0 && usage.PromptTokensDetails.AudioTokens == 0 {
		usage.PromptTokensDetails.TextTokens = usage.PromptTokens
	}
	return usage
}

// buildUpstreamUsageSnapshot 提取上游原生 usage 快照用于与供应商账单对账。
// 输入侧是本地估算（locality.PromptLocal）时返回 nil：此时连输入都对不上账单。
// 仅输出侧为估算时仍然保留快照，并标注 completion_estimated=true —— 这样
// 「输入来自上游、输出为本地估算」的请求不会丢掉已经精确拿到的上游输入 token。
func buildUpstreamUsageSnapshot(usage *dto.Usage, locality UsageLocality) map[string]interface{} {
	if usage == nil || locality.PromptLocal {
		return nil
	}
	if billingUsage := usage.BillingUsage; billingUsage != nil {
		var snap map[string]interface{}
		switch {
		case billingUsage.ClaudeUsage != nil:
			snap = upstreamUsageSnapshotClaude(billingUsage.ClaudeUsage)
		case billingUsage.GeminiUsageMetadata != nil:
			snap = upstreamUsageSnapshotGemini(billingUsage.GeminiUsageMetadata)
		case billingUsage.OpenAIUsage != nil:
			snap = upstreamUsageSnapshotOpenAI(billingUsage.OpenAIUsage)
		default:
			return nil
		}
		if snap == nil {
			return nil
		}
		snap["source"] = billingUsage.Source
		snap["semantic"] = billingUsage.Semantic
		if billingUsage.Estimated {
			snap["estimated"] = true
		}
		if locality.CompletionLocal {
			snap["completion_estimated"] = true
		}
		return snap
	}
	snap := upstreamUsageSnapshotOpenAI(usage)
	if snap != nil && locality.CompletionLocal {
		snap["completion_estimated"] = true
	}
	return snap
}

func upstreamUsageSnapshotClaude(usage *dto.ClaudeUsage) map[string]interface{} {
	if usage == nil {
		return nil
	}
	snap := map[string]interface{}{
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
	}
	putNonZero(snap, "cache_read_input_tokens", usage.CacheReadInputTokens)
	cacheCreation5m := usage.GetCacheCreation5mTokens()
	if cacheCreation5m == 0 {
		cacheCreation5m = usage.ClaudeCacheCreation5mTokens
	}
	cacheCreation1h := usage.GetCacheCreation1hTokens()
	if cacheCreation1h == 0 {
		cacheCreation1h = usage.ClaudeCacheCreation1hTokens
	}
	cacheCreationTotal := usage.CacheCreationInputTokens
	if cacheCreationTotal == 0 {
		cacheCreationTotal = cacheCreation5m + cacheCreation1h
	}
	putNonZero(snap, "cache_creation_input_tokens", cacheCreationTotal)
	if cacheCreation5m > 0 || cacheCreation1h > 0 {
		cacheCreation := map[string]interface{}{}
		putNonZero(cacheCreation, "ephemeral_5m_input_tokens", cacheCreation5m)
		putNonZero(cacheCreation, "ephemeral_1h_input_tokens", cacheCreation1h)
		snap["cache_creation"] = cacheCreation
	}
	if usage.ServerToolUse != nil && usage.ServerToolUse.WebSearchRequests > 0 {
		snap["server_tool_use"] = map[string]interface{}{
			"web_search_requests": usage.ServerToolUse.WebSearchRequests,
		}
	}
	return snap
}

func upstreamUsageSnapshotGemini(metadata *dto.GeminiUsageMetadata) map[string]interface{} {
	if metadata == nil {
		return nil
	}
	snap := map[string]interface{}{}
	putNonZero(snap, "promptTokenCount", metadata.PromptTokenCount)
	putNonZero(snap, "toolUsePromptTokenCount", metadata.ToolUsePromptTokenCount)
	putNonZero(snap, "candidatesTokenCount", metadata.CandidatesTokenCount)
	putNonZero(snap, "totalTokenCount", metadata.TotalTokenCount)
	putNonZero(snap, "thoughtsTokenCount", metadata.ThoughtsTokenCount)
	putNonZero(snap, "cachedContentTokenCount", metadata.CachedContentTokenCount)
	if len(metadata.PromptTokensDetails) > 0 {
		snap["promptTokensDetails"] = metadata.PromptTokensDetails
	}
	if len(metadata.ToolUsePromptTokensDetails) > 0 {
		snap["toolUsePromptTokensDetails"] = metadata.ToolUsePromptTokensDetails
	}
	if len(metadata.CandidatesTokensDetails) > 0 {
		snap["candidatesTokensDetails"] = metadata.CandidatesTokensDetails
	}
	return snap
}

func upstreamUsageSnapshotOpenAI(usage *dto.Usage) map[string]interface{} {
	if usage == nil {
		return nil
	}
	snap := map[string]interface{}{}
	putNonZero(snap, "prompt_tokens", usage.PromptTokens)
	putNonZero(snap, "completion_tokens", usage.CompletionTokens)
	putNonZero(snap, "total_tokens", usage.TotalTokens)
	putNonZero(snap, "prompt_cache_hit_tokens", usage.PromptCacheHitTokens)
	if details := openAIInputDetailsSnapshot(usage.PromptTokensDetails); details != nil {
		snap["prompt_tokens_details"] = details
	}
	if details := openAIOutputDetailsSnapshot(usage.CompletionTokenDetails); details != nil {
		snap["completion_tokens_details"] = details
	}
	if usage.InputTokens > 0 {
		snap["input_tokens"] = usage.InputTokens
	}
	if usage.OutputTokens > 0 {
		snap["output_tokens"] = usage.OutputTokens
	}
	if usage.InputTokensDetails != nil {
		if details := openAIInputDetailsSnapshot(*usage.InputTokensDetails); details != nil {
			snap["input_tokens_details"] = details
		}
	}
	if len(snap) == 0 {
		return nil
	}
	return snap
}

func openAIInputDetailsSnapshot(details dto.InputTokenDetails) map[string]interface{} {
	snap := map[string]interface{}{}
	putNonZero(snap, "cached_tokens", details.CachedTokens)
	putNonZero(snap, "cache_write_tokens", details.CacheWriteTokens)
	putNonZero(snap, "cached_creation_tokens", details.CachedCreationTokens)
	putNonZero(snap, "text_tokens", details.TextTokens)
	putNonZero(snap, "image_tokens", details.ImageTokens)
	putNonZero(snap, "audio_tokens", details.AudioTokens)
	if len(snap) == 0 {
		return nil
	}
	return snap
}

func openAIOutputDetailsSnapshot(details dto.OutputTokenDetails) map[string]interface{} {
	snap := map[string]interface{}{}
	putNonZero(snap, "reasoning_tokens", details.ReasoningTokens)
	putNonZero(snap, "text_tokens", details.TextTokens)
	putNonZero(snap, "image_tokens", details.ImageTokens)
	putNonZero(snap, "audio_tokens", details.AudioTokens)
	if len(snap) == 0 {
		return nil
	}
	return snap
}

func putNonZero(m map[string]interface{}, key string, value int) {
	if value != 0 {
		m[key] = value
	}
}

func addGeminiInputTokenDetail(details *dto.InputTokenDetails, detail dto.GeminiPromptTokensDetails) {
	switch detail.Modality {
	case "AUDIO":
		details.AudioTokens += detail.TokenCount
	case "IMAGE":
		details.ImageTokens += detail.TokenCount
	case "TEXT":
		details.TextTokens += detail.TokenCount
	}
}
