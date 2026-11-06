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
  parseLegalLanguageValue,
  resolveLegalLanguageValue,
  serializeLegalLanguageValue,
  updateLegalLanguageValue,
} from '../legal-language'

describe('parseLegalLanguageValue', () => {
  test('returns an empty map for blank input', () => {
    expect(parseLegalLanguageValue('')).toEqual({})
    expect(parseLegalLanguageValue('   ')).toEqual({})
    expect(parseLegalLanguageValue(null)).toEqual({})
    expect(parseLegalLanguageValue('{}')).toEqual({})
  })

  test('treats legacy plain text as the default language', () => {
    expect(parseLegalLanguageValue('# Terms')).toEqual({ '': '# Terms' })
  })

  test('parses a JSON language map and drops non-string entries', () => {
    expect(
      parseLegalLanguageValue('{"zhCN":"中文","en":"English","x":1}')
    ).toEqual({ zhCN: '中文', en: 'English' })
  })

  test('falls back to the default language for malformed JSON', () => {
    expect(parseLegalLanguageValue('{not json')).toEqual({ '': '{not json' })
  })
})

describe('resolveLegalLanguageValue', () => {
  test('prefers the exact language', () => {
    expect(
      resolveLegalLanguageValue(
        { '': 'Default', en: 'English', zhCN: '中文' },
        'zhCN'
      )
    ).toBe('中文')
  })

  test('falls back to the default language', () => {
    expect(
      resolveLegalLanguageValue({ '': 'Default', en: 'English' }, 'fr')
    ).toBe('Default')
  })

  test('falls back to English next', () => {
    expect(
      resolveLegalLanguageValue({ en: 'English', ja: '日本語' }, 'fr')
    ).toBe('English')
  })

  test('falls back to the first configured language in canonical order', () => {
    expect(
      resolveLegalLanguageValue({ vi: 'Tiếng Việt', ja: '日本語' }, 'fr')
    ).toBe('日本語')
  })

  test('ignores whitespace-only values', () => {
    expect(resolveLegalLanguageValue({ '': '  ', en: 'English' }, '')).toBe(
      'English'
    )
  })

  test('returns an empty string when nothing is configured', () => {
    expect(resolveLegalLanguageValue({}, 'en')).toBe('')
  })
})

describe('serializeLegalLanguageValue', () => {
  test('orders keys canonically and drops empty entries', () => {
    expect(
      serializeLegalLanguageValue({
        xx: 'Unknown',
        en: 'English',
        zhCN: '中文',
        '': 'Default',
        fr: '',
      })
    ).toBe('{"":"Default","zhCN":"中文","en":"English","xx":"Unknown"}')
  })

  test('keeps whitespace-only entries to avoid cursor jumps', () => {
    expect(serializeLegalLanguageValue({ en: ' ' })).toBe('{"en":" "}')
  })
})

describe('updateLegalLanguageValue', () => {
  test('updates one language and preserves the rest', () => {
    expect(
      updateLegalLanguageValue('{"":"Default","en":"Old"}', 'en', 'New')
    ).toBe('{"":"Default","en":"New"}')
  })

  test('removes a language when set to an empty string', () => {
    expect(
      updateLegalLanguageValue('{"":"Default","en":"English"}', 'en', '')
    ).toBe('{"":"Default"}')
  })

  test('migrates legacy plain text into the default language', () => {
    expect(updateLegalLanguageValue('Legacy text', 'zhCN', '中文')).toBe(
      '{"":"Legacy text","zhCN":"中文"}'
    )
  })
})
