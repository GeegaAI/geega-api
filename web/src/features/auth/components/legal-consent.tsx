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
import { Fragment, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'

import type { LegalAgreementMap } from '../hooks/use-legal-consent'
import {
  getEnabledLegalDocs,
  getLegalDocNameSeparator,
  quoteLegalDocName,
  type LegalDocKey,
} from '../lib/legal-docs'
import type { SystemStatus } from '../types'
import { LegalReadingDialog } from './legal-reading-dialog'

interface LegalConsentProps {
  status: SystemStatus | null
  agreed: LegalAgreementMap
  onAgreedChange: (key: LegalDocKey, value: boolean) => void
  className?: string
}

export function LegalConsent(props: LegalConsentProps) {
  const { t, i18n } = useTranslation()
  const lang = i18n.resolvedLanguage ?? i18n.language ?? ''
  const docs = getEnabledLegalDocs(props.status, lang)
  const [activeDocKey, setActiveDocKey] = useState<LegalDocKey | null>(null)
  const activeDoc = docs.find((doc) => doc.key === activeDocKey) ?? null

  if (docs.length === 0) {
    return null
  }

  const allAgreed = docs.every((doc) => props.agreed[doc.key] === true)
  const separator = getLegalDocNameSeparator(lang)
  const ariaLabel = t('I have carefully read and agree to')

  return (
    <div className={cn('space-y-2', props.className)}>
      <div className='flex items-start gap-3'>
        <Checkbox
          checked={allAgreed}
          aria-label={ariaLabel}
          onCheckedChange={(value) => {
            docs.forEach((doc) => props.onAgreedChange(doc.key, value === true))
          }}
          className='mt-0.5'
        />
        <Label className='text-muted-foreground items-start gap-1 text-left text-xs leading-5 font-normal'>
          <span>
            {ariaLabel}{' '}
            {docs.map((doc, index) => {
              const displayName = quoteLegalDocName(
                doc.name || t(doc.nameKey),
                lang
              )
              return (
                <Fragment key={doc.key}>
                  {index > 0 && <span>{separator}</span>}
                  <button
                    type='button'
                    onClick={() => setActiveDocKey(doc.key)}
                    className='text-primary hover:underline'
                  >
                    {displayName}
                  </button>
                </Fragment>
              )
            })}
          </span>
        </Label>
      </div>

      {activeDoc && (
        <LegalReadingDialog
          open={activeDocKey === activeDoc.key}
          onOpenChange={(open) => {
            if (!open) setActiveDocKey(null)
          }}
          doc={activeDoc}
        />
      )}
    </div>
  )
}
