package claude

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClaudeTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func newClaudeTestRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayFormat:     types.RelayFormatClaude,
		OriginModelName: "claude-3-7-sonnet",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-3-7-sonnet",
		},
	}
}

func TestHandleStreamFinalResponseKeepsUpstreamPromptWhenCompletionEstimated(t *testing.T) {
	ctx := newClaudeTestContext()
	info := newClaudeTestRelayInfo()
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{PromptTokens: 100},
		Done:  false,
	}
	claudeInfo.ResponseText.WriteString("hello world, this is a partial stream")

	HandleStreamFinalResponse(ctx, info, claudeInfo)

	// message_start 已经拿到的输入侧 token 不能因为输出侧估算了就被标成本地估算。
	require.Equal(t, 100, claudeInfo.Usage.PromptTokens)
	require.Greater(t, claudeInfo.Usage.CompletionTokens, 0)

	locality := service.UsageLocalityFromContext(ctx)
	assert.False(t, locality.PromptLocal)
	assert.True(t, locality.CompletionLocal)
	assert.Equal(t, service.UsageFallbackCompletionMissing, service.UsageFallbackReasonFromContext(ctx))
}

func TestHandleStreamFinalResponseMarksBothSidesWhenUsageMissing(t *testing.T) {
	ctx := newClaudeTestContext()
	info := newClaudeTestRelayInfo()
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{},
		Done:  false,
	}
	claudeInfo.ResponseText.WriteString("partial output")

	HandleStreamFinalResponse(ctx, info, claudeInfo)

	locality := service.UsageLocalityFromContext(ctx)
	assert.True(t, locality.PromptLocal)
	assert.True(t, locality.CompletionLocal)
	assert.Equal(t,
		service.UsageFallbackCompletionMissing+","+service.UsageFallbackPromptMissing,
		service.UsageFallbackReasonFromContext(ctx))
}

func TestHandleStreamFinalResponseDoesNotMarkLocalWhenComplete(t *testing.T) {
	ctx := newClaudeTestContext()
	info := newClaudeTestRelayInfo()
	claudeInfo := &ClaudeResponseInfo{
		Usage: &dto.Usage{PromptTokens: 100, CompletionTokens: 20},
		Done:  true,
	}

	HandleStreamFinalResponse(ctx, info, claudeInfo)

	locality := service.UsageLocalityFromContext(ctx)
	assert.False(t, locality.PromptLocal)
	assert.False(t, locality.CompletionLocal)
	assert.Empty(t, service.UsageFallbackReasonFromContext(ctx))
}
