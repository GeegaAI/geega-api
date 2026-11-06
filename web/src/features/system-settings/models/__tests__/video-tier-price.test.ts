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

import {
  buildVideoTierPriceJson,
  parseVideoTierPriceJson,
  videoTierKey,
  type VideoTierPriceEntry,
} from '../video-tier-price'

describe('videoTierKey', () => {
  test('should append _video suffix when request carries video input', () => {
    expect(videoTierKey('720p', false)).toBe('720p')
    expect(videoTierKey('720p', true)).toBe('720p_video')
  })
})

describe('parseVideoTierPriceJson', () => {
  test('should parse nested map into entries when JSON is valid', () => {
    const result = parseVideoTierPriceJson(
      '{"doubao-seedance-2-0-260128":{"720p":3.2857,"1080p_video":2.2143}}'
    )

    expect(result).toEqual({
      ok: true,
      entries: [
        {
          model: 'doubao-seedance-2-0-260128',
          prices: { '720p': '3.2857', '1080p_video': '2.2143' },
        },
      ],
    })
  })

  test('should return empty entries when input is blank or an empty object', () => {
    expect(parseVideoTierPriceJson('')).toEqual({ ok: true, entries: [] })
    expect(parseVideoTierPriceJson('{}')).toEqual({ ok: true, entries: [] })
  })

  test('should reject when JSON syntax is invalid', () => {
    expect(parseVideoTierPriceJson('{not json')).toEqual({
      ok: false,
      error: 'Invalid JSON',
    })
  })

  test('should reject when root value is not an object', () => {
    expect(parseVideoTierPriceJson('[]')).toEqual({
      ok: false,
      error: 'Expected a JSON object of model → tier prices',
    })
  })

  test('should reject when a model entry is not an object', () => {
    expect(parseVideoTierPriceJson('{"m1":5}')).toEqual({
      ok: false,
      error: 'Each model entry must be an object of tier prices',
    })
  })

  test('should reject when a tier key is unknown', () => {
    expect(parseVideoTierPriceJson('{"m1":{"144p":1}}')).toEqual({
      ok: false,
      error: 'Contains an unknown tier key',
    })
  })

  test('should reject when a price is not a positive number', () => {
    expect(parseVideoTierPriceJson('{"m1":{"720p":0}}')).toEqual({
      ok: false,
      error: 'Prices must be positive numbers',
    })
    expect(parseVideoTierPriceJson('{"m1":{"720p":"high"}}')).toEqual({
      ok: false,
      error: 'Prices must be positive numbers',
    })
  })
})

describe('buildVideoTierPriceJson', () => {
  const entry: VideoTierPriceEntry = {
    model: 'doubao-seedance-2-0-260128',
    prices: {
      '480p': '3.2857',
      '720p': '',
      '720p_video': '2',
      '1080p': ' 3.6429 ',
    },
  }

  test('should serialize configured tiers and skip blank cells', () => {
    const result = buildVideoTierPriceJson([entry])

    expect(result.ok).toBe(true)
    if (result.ok) {
      expect(JSON.parse(result.json)).toEqual({
        'doubao-seedance-2-0-260128': {
          '480p': 3.2857,
          '720p_video': 2,
          '1080p': 3.6429,
        },
      })
    }
  })

  test('should round-trip through parse when entries are valid', () => {
    const built = buildVideoTierPriceJson([entry])
    expect(built.ok).toBe(true)
    if (!built.ok) return

    const parsed = parseVideoTierPriceJson(built.json)
    expect(parsed).toEqual({
      ok: true,
      entries: [
        {
          model: 'doubao-seedance-2-0-260128',
          prices: { '480p': '3.2857', '720p_video': '2', '1080p': '3.6429' },
        },
      ],
    })
  })

  test('should return empty object when no entries remain', () => {
    expect(buildVideoTierPriceJson([])).toEqual({ ok: true, json: '{}' })
  })

  test('should reject when model name is blank', () => {
    expect(buildVideoTierPriceJson([{ model: '  ', prices: {} }])).toEqual({
      ok: false,
      error: 'Model name cannot be empty',
    })
  })

  test('should reject when model names duplicate ignoring case', () => {
    const result = buildVideoTierPriceJson([
      { model: 'Seedance', prices: {} },
      { model: 'seedance', prices: {} },
    ])

    expect(result).toEqual({ ok: false, error: 'Duplicate model name' })
  })

  test('should reject when a price cell is not positive', () => {
    expect(
      buildVideoTierPriceJson([
        { model: 'm1', prices: { '720p': '-1' } },
      ])
    ).toEqual({ ok: false, error: 'Prices must be positive numbers' })
  })
})
