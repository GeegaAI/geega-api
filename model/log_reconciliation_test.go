package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertReconciliationLog(t *testing.T, log *Log) {
	t.Helper()
	log.Type = LogTypeConsume
	log.CreatedAt = time.Now().Unix()
	require.NoError(t, LOG_DB.Create(log).Error)
}

func findReconciliationGroup(t *testing.T, groups []UsageReconciliationGroup, path string) UsageReconciliationGroup {
	t.Helper()
	for _, group := range groups {
		if group.UsageBillingPath == path {
			return group
		}
	}
	t.Fatalf("reconciliation group %q not found", path)
	return UsageReconciliationGroup{}
}

// Anthropic 上报 cache_creation 总量与 5m/1h 拆分（拆分之和=总量），
// 对账换算必须逐行二取一，不能把总量和拆分重复累计
func TestGetUsageReconciliationGroups_AnthropicCacheCreationCountedOnce(t *testing.T) {
	truncateTables(t)

	insertReconciliationLog(t, &Log{
		ChannelId:    70,
		ModelName:    "claude-sonnet-4-5",
		PromptTokens: 650,
		Other:        `{"usage_semantic":"anthropic","cache_tokens":86807,"cache_creation_tokens":24224,"cache_creation_tokens_5m":24224,"admin_info":{"usage_billing_path":"billing-usage-anthropic"}}`,
	})
	insertReconciliationLog(t, &Log{
		ChannelId:    70,
		ModelName:    "claude-sonnet-4-5",
		PromptTokens: 100,
		Other:        `{"usage_semantic":"anthropic","cache_tokens":100,"cache_creation_tokens_5m":300,"cache_creation_tokens_1h":50,"admin_info":{"usage_billing_path":"billing-usage-anthropic"}}`,
	})

	groups, err := GetUsageReconciliationGroups(1, time.Now().Unix()+60)
	require.NoError(t, err)

	group := findReconciliationGroup(t, groups, "billing-usage-anthropic")
	// 24224（总量=5m 拆分，只计一次） + 300 + 50（无总量，取拆分之和）
	assert.Equal(t, int64(24224+350), group.CacheCreationEffectiveTokens)
	assert.Equal(t, int64(750), group.PromptTokens)
	assert.Equal(t, int64(86807+100), group.CacheTokens)

	row := DeriveUsageReconciliationRow(group)
	require.True(t, row.Reconcilable)
	assert.Equal(t, int64(750+86907+24574), row.UpstreamInputTokens)
	assert.Equal(t, int64(750), row.SystemInputTokens)
}

// openai/gemini 语义的 prompt_tokens 已含缓存（缓存为其子集），上游输入即系统输入
func TestGetUsageReconciliationGroups_OpenAIUpstreamEqualsPrompt(t *testing.T) {
	truncateTables(t)

	insertReconciliationLog(t, &Log{
		ChannelId:    1,
		ModelName:    "gpt-4o",
		PromptTokens: 500,
		Other:        `{"cache_tokens":999,"admin_info":{"usage_billing_path":"billing-usage-openai"}}`,
	})

	groups, err := GetUsageReconciliationGroups(1, time.Now().Unix()+60)
	require.NoError(t, err)

	row := DeriveUsageReconciliationRow(findReconciliationGroup(t, groups, "billing-usage-openai"))
	require.True(t, row.Reconcilable)
	assert.Equal(t, int64(500), row.UpstreamInputTokens)
	assert.Equal(t, int64(500), row.SystemInputTokens)
}

// 不可对账行（未标注语义/估算/本地）上游恒为 0，避免"上游=系统"的虚假对平
func TestDeriveUsageReconciliationRow_UnreconcilableRowsHaveZeroUpstream(t *testing.T) {
	cases := []struct {
		name  string
		group UsageReconciliationGroup
	}{
		{
			name:  "legacy path without semantic",
			group: UsageReconciliationGroup{UsageBillingPath: "legacy", PromptTokens: 100, CacheTokens: 50},
		},
		{
			name:  "local estimated tokens",
			group: UsageReconciliationGroup{UsageBillingPath: "local", PromptTokens: 100},
		},
		{
			name:  "estimated anthropic",
			group: UsageReconciliationGroup{UsageBillingPath: "billing-usage-anthropic-estimated", PromptTokens: 100, CacheCreationEffectiveTokens: 40},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := DeriveUsageReconciliationRow(tc.group)
			assert.False(t, row.Reconcilable)
			assert.Equal(t, int64(0), row.UpstreamInputTokens)
		})
	}
}
