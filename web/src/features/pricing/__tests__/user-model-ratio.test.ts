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

import { SORT_OPTIONS } from '../constants'
import { sortModels } from '../lib/filters'
import { getEffectiveModelRatio, getUserModelRatio } from '../lib/model-helpers'
import { formatPrice, formatRequestPrice, formatRequestPriceOriginal } from '../lib/price'
import type { PricingModel } from '../types'

function makeModel(overrides: Partial<PricingModel>): PricingModel {
  return {
    id: 1,
    model_name: 'gpt-test',
    quota_type: 0,
    model_ratio: 2,
    completion_ratio: 3,
    enable_groups: [],
    ...overrides,
  }
}

/**
 * Extract the numeric part of a formatted price. Test prices stay below the
 * grouping threshold, so no locale uses thousands separators here; the comma
 * is only remapped for locales using it as the decimal separator.
 */
function parsePriceNumber(formatted: string): number {
  return Number.parseFloat(
    formatted.replaceAll(/[^0-9.,-]/g, '').replaceAll(',', '.')
  )
}

describe('getUserModelRatio', () => {
  it('returns undefined when the model has no override', () => {
    expect(getUserModelRatio(makeModel({}))).toBeUndefined()
  })

  it('returns the override value, preserving a configured zero', () => {
    expect(getUserModelRatio(makeModel({ user_model_ratio: 0.5 }))).toBe(0.5)
    expect(getUserModelRatio(makeModel({ user_model_ratio: 0 }))).toBe(0)
  })

  it('treats non-finite overrides as missing', () => {
    expect(
      getUserModelRatio(makeModel({ user_model_ratio: Number.NaN }))
    ).toBeUndefined()
    expect(getEffectiveModelRatio(makeModel({}))).toBe(2)
  })
})

describe('formatPrice with user model ratio override', () => {
  it('replaces the global model ratio when an override is present', () => {
    const withOverride = makeModel({ user_model_ratio: 0.5 })
    const baseline = makeModel({ model_ratio: 0.5 })
    const original = makeModel({})

    // 0.5 (override) × 2 × group 1 = same as a model whose global ratio is 0.5
    expect(formatPrice(withOverride, 'input', 'M')).toBe(
      formatPrice(baseline, 'input', 'M')
    )
    expect(formatPrice(withOverride, 'input', 'M')).not.toBe(
      formatPrice(original, 'input', 'M')
    )

    // output and cached prices build on the overridden base too
    expect(formatPrice(withOverride, 'output', 'M')).toBe(
      formatPrice(baseline, 'output', 'M')
    )
    expect(
      formatPrice(
        makeModel({ cache_ratio: 0.2, user_model_ratio: 0.5 }),
        'cache',
        'M'
      )
    ).toBe(
      formatPrice(
        makeModel({ model_ratio: 0.5, cache_ratio: 0.2 }),
        'cache',
        'M'
      )
    )
  })

  it('still multiplies the group ratio on top of the override', () => {
    const groupOverride = makeModel({
      enable_groups: ['vip'],
      group_ratio: { vip: 2 },
      user_model_ratio: 0.5,
    })

    // 0.5 (override) × 2 × 2 (group) = 2, identical to global ratio 1 × 2 × 1
    expect(formatPrice(groupOverride, 'input', 'M', false, 1, 1, 'vip')).toBe(
      formatPrice(makeModel({ model_ratio: 1 }), 'input', 'M')
    )
    expect(
      parsePriceNumber(
        formatPrice(groupOverride, 'input', 'M', false, 1, 1, 'vip')
      )
    ).toBe(2)
  })

  it('keeps the original behavior when no override matches', () => {
    const model = makeModel({})
    const explicitUndefined = makeModel({ user_model_ratio: undefined })

    expect(formatPrice(explicitUndefined, 'input', 'M')).toBe(
      formatPrice(model, 'input', 'M')
    )
    // 2 (model_ratio) × 2 = 4 USD per million input tokens
    expect(parsePriceNumber(formatPrice(model, 'input', 'M'))).toBe(4)
    // 2 × 2 × 3 (completion_ratio) = 12 USD per million output tokens
    expect(parsePriceNumber(formatPrice(model, 'output', 'M'))).toBe(12)
  })

  it('displays a zero override as free', () => {
    const free = makeModel({ user_model_ratio: 0 })

    expect(getEffectiveModelRatio(free)).toBe(0)
    const formatted = formatPrice(free, 'input', 'M')
    expect(parsePriceNumber(formatted)).toBe(0)
    expect(formatted).toBe(
      formatPrice(makeModel({ model_ratio: 0 }), 'input', 'M')
    )
  })
})

describe('per-request price with user model ratio override', () => {
  const perRequest = {
    model_name: 'image-model',
    quota_type: 1,
    model_price: 0.1,
    model_ratio: 0,
    enable_groups: ['default'],
  }

  it('multiplies the fixed price by the exclusive coefficient', () => {
    const original = parsePriceNumber(
      formatRequestPriceOriginal(makeModel(perRequest), false, 1, 1, 'default')
    )
    const discounted = parsePriceNumber(
      formatRequestPrice(
        makeModel({ ...perRequest, user_model_ratio: 0.5 }),
        false,
        1,
        1,
        'default'
      )
    )

    expect(original).toBeGreaterThan(0)
    expect(discounted).toBeCloseTo(original * 0.5, 6)
  })

  it('keeps the fixed price when there is no override', () => {
    const formatted = formatRequestPrice(
      makeModel(perRequest),
      false,
      1,
      1,
      'default'
    )
    expect(parsePriceNumber(formatted)).toBeCloseTo(
      parsePriceNumber(
        formatRequestPriceOriginal(
          makeModel(perRequest),
          false,
          1,
          1,
          'default'
        )
      ),
      6
    )
  })

  it('displays a zero coefficient as free', () => {
    const free = makeModel({ ...perRequest, user_model_ratio: 0 })
    expect(
      parsePriceNumber(formatRequestPrice(free, false, 1, 1, 'default'))
    ).toBe(0)
  })
})

describe('sortModels with user model ratio override', () => {
  it('sorts by the effective ratio so overridden prices rank correctly', () => {
    const cheap = makeModel({
      model_name: 'cheap',
      model_ratio: 2,
      user_model_ratio: 0.1,
    })
    const normal = makeModel({ model_name: 'normal', model_ratio: 1 })

    const sorted = sortModels([normal, cheap], SORT_OPTIONS.PRICE_LOW)
    expect(sorted.map((model) => model.model_name)).toEqual(['cheap', 'normal'])
  })
})
