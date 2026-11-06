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
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  getEnabledLegalDocs,
  type LegalDoc,
  type LegalDocKey,
} from '../lib/legal-docs'
import type { SystemStatus } from '../types'

export type LegalAgreementMap = Partial<Record<LegalDocKey, boolean>>

export type LegalConsentState = {
  docs: LegalDoc[]
  requiresConsent: boolean
  agreed: LegalAgreementMap
  allAgreed: boolean
  setDocAgreed: (key: LegalDocKey, value: boolean) => void
}

export function useLegalConsent(
  status: SystemStatus | null
): LegalConsentState {
  const { i18n } = useTranslation()
  const lang = i18n.resolvedLanguage ?? i18n.language ?? ''
  const docs = useMemo(() => getEnabledLegalDocs(status, lang), [status, lang])
  const [agreed, setAgreed] = useState<LegalAgreementMap>({})

  const docKeys = docs.map((doc) => doc.key).join(',')
  useEffect(() => {
    setAgreed({})
  }, [docKeys])

  const setDocAgreed = useCallback((key: LegalDocKey, value: boolean) => {
    setAgreed((prev) => ({ ...prev, [key]: value }))
  }, [])

  const requiresConsent = docs.length > 0
  const allAgreed = docs.every((doc) => agreed[doc.key] === true)

  return { docs, requiresConsent, agreed, allAgreed, setDocAgreed }
}
