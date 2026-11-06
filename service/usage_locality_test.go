package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func TestUsageLocalityFromContextLegacyFlag(t *testing.T) {
	ctx := newTestContext(t)
	common.SetContextKey(ctx, constant.ContextKeyLocalCountTokens, true)

	locality := UsageLocalityFromContext(ctx)
	require.True(t, locality.PromptLocal)
	require.True(t, locality.CompletionLocal)
	require.True(t, locality.AllLocal())
}

func TestUsageLocalityFromContextPrefersSplitFlags(t *testing.T) {
	ctx := newTestContext(t)
	// 新标记存在时，不应被历史总开关反向覆盖成"两侧都本地"。
	common.SetContextKey(ctx, constant.ContextKeyLocalCountTokens, true)
	common.SetContextKey(ctx, constant.ContextKeyLocalCompletionTokens, true)

	locality := UsageLocalityFromContext(ctx)
	require.False(t, locality.PromptLocal)
	require.True(t, locality.CompletionLocal)
	require.True(t, locality.AnyLocal())
}

func TestMarkPromptTokensLocalOnlyMarksPrompt(t *testing.T) {
	ctx := newTestContext(t)
	MarkPromptTokensLocal(ctx, UsageFallbackPromptMissing)

	locality := UsageLocalityFromContext(ctx)
	require.True(t, locality.PromptLocal)
	require.False(t, locality.CompletionLocal)
	require.Equal(t, UsageFallbackPromptMissing, UsageFallbackReasonFromContext(ctx))
	// 保留历史总开关，兼容旧日志/前端判断。
	require.True(t, common.GetContextKeyBool(ctx, constant.ContextKeyLocalCountTokens))
}

func TestMarkAllTokensLocalRecordsBothSides(t *testing.T) {
	ctx := newTestContext(t)
	MarkAllTokensLocal(ctx, UsageFallbackUsageMissing)

	locality := UsageLocalityFromContext(ctx)
	require.True(t, locality.PromptLocal)
	require.True(t, locality.CompletionLocal)
	require.Equal(t, UsageFallbackUsageMissing, UsageFallbackReasonFromContext(ctx))
}

func TestUsageFallbackReasonDeduplicatesPerRequest(t *testing.T) {
	ctx := newTestContext(t)
	MarkCompletionTokensLocal(ctx, UsageFallbackCompletionMissing)
	MarkCompletionTokensLocal(ctx, UsageFallbackCompletionMissing)
	MarkAllTokensLocal(ctx, UsageFallbackStreamUsageMissing)

	reason := UsageFallbackReasonFromContext(ctx)
	assert.Equal(t, UsageFallbackCompletionMissing+","+UsageFallbackStreamUsageMissing, reason)
}

func TestUsageFallbackStatsCountOncePerReasonPerRequest(t *testing.T) {
	ResetUsageFallbackStats()
	t.Cleanup(ResetUsageFallbackStats)

	first := newTestContext(t)
	MarkCompletionTokensLocal(first, UsageFallbackCompletionMissing)
	MarkCompletionTokensLocal(first, UsageFallbackCompletionMissing) // 同一请求同一原因只计一次

	second := newTestContext(t)
	MarkCompletionTokensLocal(second, UsageFallbackCompletionMissing)

	stats := GetUsageFallbackStats()
	assert.EqualValues(t, 2, stats[UsageFallbackCompletionMissing])
}

func TestResetUsageLocalityClearsFlagsAndReasons(t *testing.T) {
	ctx := newTestContext(t)
	MarkAllTokensLocal(ctx, UsageFallbackStreamUsageMissing)
	require.True(t, UsageLocalityFromContext(ctx).AnyLocal())
	require.NotEmpty(t, UsageFallbackReasonFromContext(ctx))

	ResetUsageLocality(ctx)

	require.False(t, UsageLocalityFromContext(ctx).AnyLocal())
	require.Empty(t, UsageFallbackReasonFromContext(ctx))
	require.False(t, common.GetContextKeyBool(ctx, constant.ContextKeyLocalCountTokens))
}

func TestBuildUpstreamUsageSnapshotKeepsUpstreamInputWhenCompletionEstimated(t *testing.T) {
	usage := &dto.Usage{
		BillingUsage: dto.NewClaudeMessagesBillingUsage(&dto.ClaudeUsage{
			InputTokens:  120,
			OutputTokens: 30,
		}),
	}

	snapshot := buildUpstreamUsageSnapshot(usage, UsageLocality{CompletionLocal: true})
	require.NotNil(t, snapshot)
	assert.EqualValues(t, 120, snapshot["input_tokens"])
	assert.EqualValues(t, 30, snapshot["output_tokens"])
	assert.Equal(t, true, snapshot["completion_estimated"])
	assert.Equal(t, "claude_messages", snapshot["source"])
}

func TestBuildUpstreamUsageSnapshotOmitsWhenPromptEstimated(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
	}

	require.Nil(t, buildUpstreamUsageSnapshot(usage, UsageLocality{PromptLocal: true}))
	require.Nil(t, buildUpstreamUsageSnapshot(usage, UsageLocality{PromptLocal: true, CompletionLocal: true}))
}

func TestBuildUpstreamUsageSnapshotOpenAIClaudeLocalOnly(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
	}

	snapshot := buildUpstreamUsageSnapshot(usage, UsageLocality{})
	require.NotNil(t, snapshot)
	assert.EqualValues(t, 100, snapshot["prompt_tokens"])
	assert.NotContains(t, snapshot, "completion_estimated")

	snapshot = buildUpstreamUsageSnapshot(usage, UsageLocality{CompletionLocal: true})
	require.NotNil(t, snapshot)
	assert.Equal(t, true, snapshot["completion_estimated"])
}
