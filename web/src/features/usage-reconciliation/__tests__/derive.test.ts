/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, it } from 'vitest'

import {
  aggregateReconciliationRowsByPath,
  computeReconcilableShare,
  computeRowInputDelta,
  formatPercent,
  formatTokenCount,
  groupReconciliationRowsByChannel,
  isReconcilable,
} from '../lib/derive'
import type { UsageReconciliationRow } from '../types'

function makeRow(overrides: Partial<UsageReconciliationRow>): UsageReconciliationRow {
  return {
    usage_billing_path: 'billing-usage-anthropic',
    semantic: 'anthropic',
    estimated: false,
    local: false,
    reconcilable: true,
    channel_id: 0,
    channel_name: '',
    model_name: '',
    request_count: 1,
    quota: 0,
    system_input_tokens: 0,
    system_output_tokens: 0,
    upstream_input_tokens: 0,
    ...overrides,
  }
}

describe('computeRowInputDelta', () => {
  it('returns positive diff when upstream counts more tokens than system', () => {
    const row = makeRow({ system_input_tokens: 1000, upstream_input_tokens: 3500 })
    expect(computeRowInputDelta(row)).toBe(2500)
  })

  it('returns negative diff when system counts more tokens than upstream', () => {
    const row = makeRow({ system_input_tokens: 2000, upstream_input_tokens: 1800 })
    expect(computeRowInputDelta(row)).toBe(-200)
  })
})

describe('isReconcilable', () => {
  it('follows the backend reconcilable flag instead of guessing', () => {
    expect(isReconcilable(makeRow({}))).toBe(true)
    expect(isReconcilable(makeRow({ reconcilable: false }))).toBe(false)
    expect(isReconcilable(makeRow({ reconcilable: false, estimated: true }))).toBe(false)
    expect(isReconcilable(makeRow({ reconcilable: false, local: true }))).toBe(false)
  })
})

describe('computeReconcilableShare', () => {
  it('returns 0 for empty groups', () => {
    expect(computeReconcilableShare([])).toBe(0)
  })

  it('returns share of reconcilable requests over total requests', () => {
    const rows = [
      makeRow({ request_count: 7 }),
      makeRow({ reconcilable: false, local: true, request_count: 2 }),
      makeRow({ reconcilable: false, estimated: true, request_count: 1 }),
    ]
    expect(computeReconcilableShare(rows)).toBeCloseTo(0.7)
  })

  it('returns 0 when no request exists to avoid division by zero', () => {
    expect(computeReconcilableShare([makeRow({ request_count: 0 })])).toBe(0)
  })
})

describe('formatting', () => {
  it('formats token counts with thousand separators', () => {
    expect(formatTokenCount(1234567)).toBe('1,234,567')
  })

  it('formats fractions as percent with one decimal', () => {
    expect(formatPercent(0.1234)).toBe('12.3%')
    expect(formatPercent(0)).toBe('0.0%')
  })
})

describe('aggregateReconciliationRowsByPath', () => {
  it('merges rows sharing the same path and semantic by summing all metrics', () => {
    const merged = aggregateReconciliationRowsByPath([
      makeRow({ channel_id: 1601, model_name: 'claude-3', request_count: 2, quota: 100, system_input_tokens: 10, upstream_input_tokens: 20 }),
      makeRow({ channel_id: 1602, model_name: 'claude-4', request_count: 3, quota: 50, system_input_tokens: 5, upstream_input_tokens: 9 }),
    ])
    expect(merged).toHaveLength(1)
    expect(merged[0].usage_billing_path).toBe('billing-usage-anthropic')
    expect(merged[0].request_count).toBe(5)
    expect(merged[0].quota).toBe(150)
    expect(merged[0].system_input_tokens).toBe(15)
    expect(merged[0].upstream_input_tokens).toBe(29)
  })

  it('keeps different paths or semantics as separate rows', () => {
    const merged = aggregateReconciliationRowsByPath([
      makeRow({ usage_billing_path: 'billing-usage-anthropic', semantic: 'anthropic', quota: 10 }),
      makeRow({ usage_billing_path: 'billing-usage-openai', semantic: 'openai', quota: 20 }),
      makeRow({ usage_billing_path: 'legacy', semantic: 'anthropic', quota: 30 }),
    ])
    expect(merged).toHaveLength(3)
    expect(merged.map((row) => row.quota)).toEqual([30, 20, 10])
  })

  it('sorts merged rows by quota descending', () => {
    const merged = aggregateReconciliationRowsByPath([
      makeRow({ usage_billing_path: 'a', semantic: 'x', quota: 5 }),
      makeRow({ usage_billing_path: 'b', semantic: 'y', quota: 50 }),
      makeRow({ usage_billing_path: 'c', semantic: 'z', quota: 500 }),
    ])
    expect(merged.map((row) => row.usage_billing_path)).toEqual(['c', 'b', 'a'])
  })

  it('marks a merged row unreconcilable when any child is not reconcilable', () => {
    const merged = aggregateReconciliationRowsByPath([
      makeRow({ quota: 5 }),
      makeRow({ reconcilable: false, quota: 3 }),
    ])
    expect(merged).toHaveLength(1)
    expect(merged[0].reconcilable).toBe(false)
  })
})

describe('groupReconciliationRowsByChannel', () => {
  it('groups rows by channel id and sums metrics with flags OR-ed', () => {
    const groups = groupReconciliationRowsByChannel([
      makeRow({ channel_id: 1601, channel_name: '西部渠道', request_count: 1, quota: 10, system_input_tokens: 1, upstream_input_tokens: 2 }),
      makeRow({ channel_id: 1601, channel_name: '西部渠道', estimated: true, reconcilable: false, request_count: 4, quota: 20, system_input_tokens: 3, upstream_input_tokens: 0 }),
      makeRow({ channel_id: 1602, channel_name: 'OpenAI 官方', request_count: 2, quota: 40 }),
    ])
    expect(groups).toHaveLength(2)
    expect(groups.map((group) => group.channelId)).toEqual([1602, 1601])
    const west = groups.find((group) => group.channelId === 1601)
    expect(west?.channelName).toBe('西部渠道')
    expect(west?.requestCount).toBe(5)
    expect(west?.quota).toBe(30)
    expect(west?.systemInputTokens).toBe(4)
    expect(west?.reconciledSystemInputTokens).toBe(1)
    expect(west?.upstreamInputTokens).toBe(2)
    expect(west?.unreconciledRequests).toBe(4)
    expect(west?.estimated).toBe(true)
    expect(west?.local).toBe(false)
  })

  it('scopes upstream totals to reconcilable children only', () => {
    const groups = groupReconciliationRowsByChannel([
      makeRow({ channel_id: 1, channel_name: 'A', request_count: 3, system_input_tokens: 100, upstream_input_tokens: 150 }),
      makeRow({ channel_id: 1, channel_name: 'A', reconcilable: false, request_count: 9, system_input_tokens: 900 }),
    ])
    const group = groups[0]
    expect(group.requestCount).toBe(12)
    expect(group.unreconciledRequests).toBe(9)
    expect(group.upstreamInputTokens).toBe(150)
    expect(group.reconciledSystemInputTokens).toBe(100)
    expect(group.systemInputTokens).toBe(1000)
  })

  it('sorts groups and their model children by quota descending', () => {
    const groups = groupReconciliationRowsByChannel([
      makeRow({ channel_id: 1, model_name: 'm-small', quota: 1 }),
      makeRow({ channel_id: 1, model_name: 'm-big', quota: 9 }),
      makeRow({ channel_id: 2, model_name: 'm-mid', quota: 5 }),
    ])
    expect(groups.map((group) => group.channelId)).toEqual([1, 2])
    expect(groups[0].children.map((row) => row.model_name)).toEqual(['m-big', 'm-small'])
  })

  it('keeps rows without a channel under channel id 0', () => {
    const groups = groupReconciliationRowsByChannel([makeRow({ channel_id: 0, channel_name: '' })])
    expect(groups).toHaveLength(1)
    expect(groups[0].channelId).toBe(0)
    expect(groups[0].channelName).toBe('')
  })
})
