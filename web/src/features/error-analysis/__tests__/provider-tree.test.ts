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

import { buildErrorProviderTree } from '../lib/provider-tree'
import type { ErrorBreakdownRow, ErrorChannelStat } from '../types'

function makeChannel(overrides: Partial<ErrorChannelStat>): ErrorChannelStat {
  return {
    channel_id: 1,
    channel_name: 'ch-1',
    channel_type: 1,
    error_count: 0,
    request_count: 0,
    error_rate: 0,
    top_reason_code: '',
    top_reason_count: 0,
    model_count: 0,
    user_count: 0,
    last_error_at: 0,
    ...overrides,
  }
}

function makeRow(overrides: Partial<ErrorBreakdownRow>): ErrorBreakdownRow {
  return {
    channel_id: 1,
    channel_name: 'ch-1',
    model_name: 'gpt-x',
    reason_code: 'other',
    error_count: 1,
    reason_ratio: 0,
    channel_model_request_count: 10,
    channel_model_error_count: 1,
    channel_model_error_rate: 0.1,
    status_codes: [500],
    user_count: 1,
    sample_message: 'boom',
    ...overrides,
  }
}

describe('buildErrorProviderTree', () => {
  it('groups channels by channel type and aggregates provider-level metrics', () => {
    const tree = buildErrorProviderTree(
      [
        makeChannel({
          channel_id: 1,
          error_count: 3,
          request_count: 10,
          last_error_at: 100,
        }),
        makeChannel({
          channel_id: 2,
          error_count: 7,
          request_count: 20,
          last_error_at: 200,
        }),
      ],
      []
    )

    expect(tree).toHaveLength(1)
    expect(tree[0].type_label).toBe('OpenAI')
    expect(tree[0].error_count).toBe(10)
    expect(tree[0].request_count).toBe(30)
    expect(tree[0].error_rate).toBeCloseTo(10 / 30)
    expect(tree[0].last_error_at).toBe(200)
  })

  it('falls back to the unknown group for unmapped channel types', () => {
    const tree = buildErrorProviderTree(
      [makeChannel({ channel_type: 999 })],
      []
    )

    expect(tree).toHaveLength(1)
    expect(tree[0].key).toBe('unknown')
    expect(tree[0].known_type).toBe(false)
  })

  it('builds one row per model with top reason, unioned status codes and its sample message', () => {
    const tree = buildErrorProviderTree(
      [makeChannel({ channel_id: 1 })],
      [
        makeRow({
          model_name: 'gpt-x',
          reason_code: 'timeout',
          error_count: 2,
          status_codes: [504],
          sample_message: 'timeout body',
        }),
        makeRow({
          model_name: 'gpt-x',
          reason_code: 'rate_limited',
          error_count: 5,
          status_codes: [429, 504],
          sample_message: 'rate limited body',
        }),
        makeRow({
          model_name: 'other-model',
          reason_code: 'other',
          error_count: 1,
          status_codes: [500],
          sample_message: 'other body',
        }),
      ]
    )

    const models = tree[0].channels[0].models
    expect(models).toHaveLength(2)
    expect(models[0].model_name).toBe('gpt-x')
    expect(models[0].error_count).toBe(7)
    expect(models[0].top_reason_code).toBe('rate_limited')
    expect(models[0].sample_message).toBe('rate limited body')
    expect(models[0].status_codes).toEqual([429, 504])
  })

  it('sorts providers by error count descending', () => {
    const tree = buildErrorProviderTree(
      [
        makeChannel({ channel_id: 1, channel_type: 1, error_count: 2 }),
        makeChannel({ channel_id: 2, channel_type: 3, error_count: 9 }),
      ],
      []
    )

    expect(tree.map((group) => group.channel_type)).toEqual([3, 1])
  })
})
