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
import { describe, expect, test } from 'vitest'

import type { PricingModel } from '@/features/pricing/types'

import {
  buildModelRatioRows,
  buildRatiosPayload,
  parseRatioDraft,
} from '../lib/user-model-ratios'

function makeModel(
  overrides: Partial<PricingModel> & { model_name: string }
): PricingModel {
  return {
    id: 1,
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    ...overrides,
  }
}

describe('parseRatioDraft', () => {
  test('empty or whitespace-only input means no discount (inherit global price)', () => {
    expect(parseRatioDraft('')).toEqual({ kind: 'inherit' })
    expect(parseRatioDraft('   ')).toEqual({ kind: 'inherit' })
  })

  test('a finite number in [0, 1) parses as valid, including 0 for free usage', () => {
    expect(parseRatioDraft('0.5')).toEqual({ kind: 'valid', ratio: 0.5 })
    expect(parseRatioDraft(' 0 ')).toEqual({ kind: 'valid', ratio: 0 })
  })

  test('negative, non-numeric, or values >= 1 are invalid', () => {
    expect(parseRatioDraft('-1')).toEqual({ kind: 'invalid' })
    expect(parseRatioDraft('abc')).toEqual({ kind: 'invalid' })
    expect(parseRatioDraft('NaN')).toEqual({ kind: 'invalid' })
    expect(parseRatioDraft('1')).toEqual({ kind: 'invalid' })
    expect(parseRatioDraft('1.5')).toEqual({ kind: 'invalid' })
  })
})

describe('buildRatiosPayload', () => {
  test('drops empty drafts so cleared rows go back to no discount', () => {
    expect(buildRatiosPayload({ 'gpt-a': '', 'gpt-b': '0.5' })).toEqual({
      'gpt-b': 0.5,
    })
  })

  test('returns an empty map when every draft is cleared (clear-all semantics)', () => {
    expect(buildRatiosPayload({ 'gpt-a': '', 'gpt-b': ' ' })).toEqual({})
    expect(buildRatiosPayload({})).toEqual({})
  })

  test('returns null when any draft is invalid so saving is blocked', () => {
    expect(buildRatiosPayload({ 'gpt-a': '0.9' })).toEqual({ 'gpt-a': 0.9 })
    expect(buildRatiosPayload({ 'gpt-a': '-0.5' })).toBeNull()
    expect(buildRatiosPayload({ 'gpt-a': '1' })).toBeNull()
    expect(buildRatiosPayload({ 'gpt-a': '0.9', 'gpt-b': 'oops' })).toBeNull()
  })
})

describe('buildModelRatioRows', () => {
  const models = [
    makeModel({ model_name: 'gpt-b', enable_groups: ['default'] }),
    makeModel({ model_name: 'gpt-a', enable_groups: ['default', 'vip'] }),
    makeModel({ model_name: 'vip-only', enable_groups: ['vip'] }),
    makeModel({ model_name: 'per-call', quota_type: 1, model_price: 0.1 }),
    makeModel({
      model_name: 'tiered-model',
      billing_mode: 'tiered_expr',
    }),
  ]

  test('lists only models enabled for the target group, sorted by name', () => {
    const rows = buildModelRatioRows(models, 'default', {})
    expect(rows.map((row) => row.modelName)).toEqual([
      'gpt-a',
      'gpt-b',
      'per-call',
      'tiered-model',
    ])
  })

  test('flags rows that already have a per-user override', () => {
    const rows = buildModelRatioRows(models, 'default', { 'gpt-b': 0.5 })
    const byName = Object.fromEntries(rows.map((row) => [row.modelName, row]))
    expect(byName['gpt-b'].overridden).toBe(true)
    expect(byName['gpt-a'].overridden).toBe(false)
  })

  test('marks models without any configured pricing', () => {
    const rows = buildModelRatioRows(
      [
        makeModel({
          model_name: 'unconfigured',
          enable_groups: ['default'],
          ratio_configured: false,
        }),
        makeModel({ model_name: 'configured', enable_groups: ['default'] }),
      ],
      'default',
      {}
    )
    const byName = Object.fromEntries(rows.map((row) => [row.modelName, row]))
    expect(byName['unconfigured'].configured).toBe(false)
    expect(byName['configured'].configured).toBe(true)
  })

  test('carries billing_mode through to rows', () => {
    const rows = buildModelRatioRows(models, 'default', {})
    const byName = Object.fromEntries(rows.map((row) => [row.modelName, row]))
    expect(byName['tiered-model'].billingMode).toBe('tiered_expr')
    expect(byName['gpt-a'].billingMode).toBe('')
  })

  test('carries the fixed per-request price so the editor can show it', () => {
    const rows = buildModelRatioRows(models, 'default', {})
    const byName = Object.fromEntries(rows.map((row) => [row.modelName, row]))
    expect(byName['per-call'].modelPrice).toBe(0.1)
    expect(byName['per-call'].quotaType).toBe(1)
  })
})
