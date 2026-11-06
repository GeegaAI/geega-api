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
import {
  resolveLegalLanguageValue,
  type LegalLanguageValueMap,
} from '@/features/legal/legal-language'

import type { SystemStatus } from '../types'

export type LegalDocKey =
  | 'user_agreement'
  | 'privacy_policy'
  | 'service_agreement'

export type LegalDoc = {
  key: LegalDocKey
  href: string
  apiPath: string
  name: string
  nameKey: string
}

type LegalDocDefinition = LegalDoc & {
  enabled: boolean
}

const LEGAL_DOC_DEFINITIONS: Array<
  Omit<LegalDocDefinition, 'name' | 'enabled'>
> = [
  {
    key: 'user_agreement',
    href: '/user-agreement',
    apiPath: '/api/user-agreement',
    nameKey: 'User Agreement',
  },
  {
    key: 'privacy_policy',
    href: '/privacy-policy',
    apiPath: '/api/privacy-policy',
    nameKey: 'Privacy Policy',
  },
  {
    key: 'service_agreement',
    href: '/service-agreement',
    apiPath: '/api/service-agreement',
    nameKey: 'Product Service Agreement',
  },
]

function statusValue(status: SystemStatus | null, key: string): unknown {
  const direct = (status as Record<string, unknown> | null)?.[key]
  if (direct !== undefined) return direct
  return ((status?.data ?? {}) as Record<string, unknown>)[key]
}

function getLegalDocNameMap(
  status: SystemStatus | null,
  key: LegalDocKey
): LegalLanguageValueMap {
  const names = statusValue(status, 'legal_doc_names')
  if (names && typeof names === 'object' && !Array.isArray(names)) {
    const map = (names as Record<string, unknown>)[key]
    if (map && typeof map === 'object' && !Array.isArray(map)) {
      return map as LegalLanguageValueMap
    }
  }
  // 兼容旧版 status（本地缓存的旧数据）与仅配置单一名称的情况
  const legacyName = String(statusValue(status, `${key}_name`) ?? '')
  return legacyName.trim() ? { '': legacyName } : {}
}

export function getLegalDocDisplayName(
  status: SystemStatus | null,
  key: LegalDocKey,
  lang: string
): string {
  return resolveLegalLanguageValue(getLegalDocNameMap(status, key), lang).trim()
}

export function getEnabledLegalDocs(
  status: SystemStatus | null,
  lang = ''
): LegalDoc[] {
  return LEGAL_DOC_DEFINITIONS.filter((definition) =>
    Boolean(statusValue(status, `${definition.key}_enabled`))
  ).map((definition) => ({
    ...definition,
    name: getLegalDocDisplayName(status, definition.key, lang),
  }))
}

const CJK_LANGUAGES = new Set(['zhCN', 'zhTW', 'ja'])

// 《》书名号仅 CJK 界面语言加，en/fr/ru/vi 不加（英文排版惯例）
export function quoteLegalDocName(name: string, lang: string): string {
  return CJK_LANGUAGES.has(lang) ? `《${name}》` : name
}

export function getLegalDocNameSeparator(lang: string): string {
  return CJK_LANGUAGES.has(lang) ? '、' : ', '
}
