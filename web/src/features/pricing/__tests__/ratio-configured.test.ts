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
import i18next from 'i18next'
import { beforeAll, describe, expect, it } from 'vitest'

import { normalizeServerBaseUrl } from '../lib/model-helpers'
import { formatBasePriceOriginal, formatPrice } from '../lib/price'
import type { PricingModel } from '../types'

function makeModel(overrides: Partial<PricingModel>): PricingModel {
  return {
    id: 1,
    model_name: 'unconfigured-model',
    quota_type: 0,
    model_ratio: 37.5,
    completion_ratio: 1,
    enable_groups: [],
    ...overrides,
  }
}

describe('ratio_configured', () => {
  beforeAll(async () => {
    await i18next.init({
      lng: 'en',
      fallbackLng: 'en',
      interpolation: { escapeValue: false },
    })
    i18next.addResourceBundle('en', 'translation', {
      'Not configured': 'Not configured',
    })
  })

  it('未配置定价的模型显示未配置而不是兜底倍率价格', () => {
    const model = makeModel({ ratio_configured: false })

    expect(formatBasePriceOriginal(model, 'input', 'M')).toBe('Not configured')
  })

  it('未配置但用户有专属倍率时，专属价格可正常计算', () => {
    const model = makeModel({ ratio_configured: false, user_model_ratio: 0.5 })

    expect(formatPrice(model, 'input', 'M')).not.toBe('Not configured')
  })

  it('未配置时全局基准价仍显示未配置（即使有专属倍率）', () => {
    const model = makeModel({ ratio_configured: false, user_model_ratio: 0.5 })

    expect(formatBasePriceOriginal(model, 'input', 'M')).toBe('Not configured')
  })

  it('已配置定价的模型仍按倍率计算价格', () => {
    const model = makeModel({ ratio_configured: true })

    expect(formatBasePriceOriginal(model, 'input', 'M')).not.toBe(
      'Not configured'
    )
  })
})

describe('normalizeServerBaseUrl', () => {
  it('去掉尾部斜杠与 /v1 后缀，避免示例出现 /v1/v1', () => {
    const cases: Array<[string, string]> = [
      ['https://tokenhub.example.com/v1', 'https://tokenhub.example.com'],
      ['https://tokenhub.example.com/v1/', 'https://tokenhub.example.com'],
      ['https://tokenhub.example.com/', 'https://tokenhub.example.com'],
      ['https://tokenhub.example.com', 'https://tokenhub.example.com'],
    ]
    for (const [input, expected] of cases) {
      expect(normalizeServerBaseUrl(input)).toBe(expected)
    }
  })
})
