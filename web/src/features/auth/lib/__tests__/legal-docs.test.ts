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

import type { SystemStatus } from '../../types'
import {
  getEnabledLegalDocs,
  getLegalDocNameSeparator,
  quoteLegalDocName,
} from '../legal-docs'

function makeStatus(overrides: Record<string, unknown>): SystemStatus {
  return overrides as SystemStatus
}

describe('getEnabledLegalDocs', () => {
  test('returns only enabled documents with configured names and paths', () => {
    const docs = getEnabledLegalDocs(
      makeStatus({
        user_agreement_enabled: true,
        user_agreement_name: '示例平台用户协议',
        privacy_policy_enabled: true,
        service_agreement_enabled: true,
        service_agreement_name: '示例平台产品服务协议',
      })
    )

    expect(docs.map((doc) => doc.key)).toEqual([
      'user_agreement',
      'privacy_policy',
      'service_agreement',
    ])
    expect(docs[0].name).toBe('示例平台用户协议')
    expect(docs[0].href).toBe('/user-agreement')
    expect(docs[0].apiPath).toBe('/api/user-agreement')
    expect(docs[1].name).toBe('')
    expect(docs[2].apiPath).toBe('/api/service-agreement')
  })

  test('returns an empty list when no document is enabled', () => {
    const docs = getEnabledLegalDocs(
      makeStatus({
        user_agreement_enabled: false,
        privacy_policy_enabled: false,
        service_agreement_enabled: false,
      })
    )
    expect(docs).toEqual([])
  })

  test('reads flags from the nested data payload as a fallback', () => {
    const docs = getEnabledLegalDocs(
      makeStatus({ data: { privacy_policy_enabled: true } })
    )
    expect(docs.map((doc) => doc.key)).toEqual(['privacy_policy'])
  })
})

describe('getEnabledLegalDocs per-language names', () => {
  const multilingualStatus = makeStatus({
    user_agreement_enabled: true,
    privacy_policy_enabled: true,
    legal_doc_names: {
      user_agreement: {
        '': 'Default Agreement',
        zhCN: '中文协议',
        en: 'English Agreement',
      },
      privacy_policy: { en: 'English Privacy' },
    },
  })

  test('uses the display name of the requested language', () => {
    const docs = getEnabledLegalDocs(multilingualStatus, 'zhCN')
    expect(docs[0].name).toBe('中文协议')
  })

  test('falls back to the default name and then English', () => {
    const docs = getEnabledLegalDocs(multilingualStatus, 'fr')
    expect(docs[0].name).toBe('Default Agreement')
    expect(docs[1].name).toBe('English Privacy')
  })

  test('still reads the legacy flat name when no language map exists', () => {
    const docs = getEnabledLegalDocs(
      makeStatus({
        user_agreement_enabled: true,
        user_agreement_name: 'Legacy Name',
      }),
      'ja'
    )
    expect(docs[0].name).toBe('Legacy Name')
  })
})

describe('quoteLegalDocName', () => {
  test('wraps the name in CJK title brackets for CJK interface languages', () => {
    expect(quoteLegalDocName('用户协议', 'zhCN')).toBe('《用户协议》')
    expect(quoteLegalDocName('用戶協議', 'zhTW')).toBe('《用戶協議》')
    expect(quoteLegalDocName('利用規約', 'ja')).toBe('《利用規約》')
  })

  test('returns the plain name for non-CJK interface languages', () => {
    expect(quoteLegalDocName('User Agreement', 'en')).toBe('User Agreement')
    expect(quoteLegalDocName('Accord', 'fr')).toBe('Accord')
    expect(quoteLegalDocName('Соглашение', 'ru')).toBe('Соглашение')
  })
})

describe('getLegalDocNameSeparator', () => {
  test('uses the CJK enumeration comma for CJK interface languages', () => {
    expect(getLegalDocNameSeparator('zhCN')).toBe('、')
    expect(getLegalDocNameSeparator('ja')).toBe('、')
  })

  test('uses a comma separator for non-CJK interface languages', () => {
    expect(getLegalDocNameSeparator('en')).toBe(', ')
    expect(getLegalDocNameSeparator('vi')).toBe(', ')
  })
})
