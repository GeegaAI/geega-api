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
import { INTERFACE_LANGUAGE_OPTIONS } from '@/i18n/languages'

// 默认内容用于所有语言都未单独配置时的兜底，与后端 LegalFallbackLanguage 对应。
export const LEGAL_DEFAULT_LANGUAGE = ''

export const LEGAL_LANGUAGE_CODES = INTERFACE_LANGUAGE_OPTIONS.map(
  (option) => option.code
)

export type LegalLanguageValueMap = Record<string, string>

const LEGAL_LANGUAGE_ORDER = [LEGAL_DEFAULT_LANGUAGE, ...LEGAL_LANGUAGE_CODES]

/**
 * 解析按语言存储的字段。历史数据是纯文本，视为默认语言内容。
 */
export function parseLegalLanguageValue(
  raw: string | null | undefined
): LegalLanguageValueMap {
  const trimmed = (raw ?? '').trim()
  if (trimmed === '') return {}
  if (trimmed.startsWith('{')) {
    try {
      const parsed: unknown = JSON.parse(trimmed)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        const result: LegalLanguageValueMap = {}
        for (const [key, value] of Object.entries(parsed)) {
          if (typeof value === 'string') result[key] = value
        }
        return result
      }
    } catch {
      // 不是合法 JSON，按历史纯文本处理
    }
  }
  return { [LEGAL_DEFAULT_LANGUAGE]: raw ?? '' }
}

/**
 * 与后端一致的取值顺序：指定语言 -> 默认 -> 英文 -> 其余语言。
 */
export function resolveLegalLanguageValue(
  values: LegalLanguageValueMap,
  lang: string
): string {
  const hasValue = (key: string) => (values[key] ?? '').trim() !== ''
  if (lang && hasValue(lang)) return values[lang]
  for (const key of [LEGAL_DEFAULT_LANGUAGE, 'en', ...LEGAL_LANGUAGE_CODES]) {
    if (hasValue(key)) return values[key]
  }
  return ''
}

/**
 * 以固定顺序序列化，保证表单默认值与保存结果一致，避免误判为已修改。
 */
export function serializeLegalLanguageValue(
  values: LegalLanguageValueMap
): string {
  const keys = LEGAL_LANGUAGE_ORDER.filter((key) => key in values)
  for (const key of Object.keys(values)) {
    if (!keys.includes(key)) keys.push(key)
  }
  const ordered: LegalLanguageValueMap = {}
  for (const key of keys) {
    const value = values[key]
    if (typeof value === 'string' && value !== '') ordered[key] = value
  }
  return JSON.stringify(ordered)
}

export function updateLegalLanguageValue(
  raw: string,
  lang: string,
  value: string
): string {
  return serializeLegalLanguageValue({
    ...parseLegalLanguageValue(raw),
    [lang]: value,
  })
}
