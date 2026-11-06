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
import { useQuery } from '@tanstack/react-query'
import { FileWarning } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import type { LegalDocumentResponse } from '@/features/legal/types'
import { api } from '@/lib/api'
import { isHttpUrl, isLikelyHtml } from '@/lib/content-format'

import type { LegalDoc } from '../lib/legal-docs'

type LegalReadingDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  doc: LegalDoc
}

async function fetchLegalDocument(apiPath: string, lang: string) {
  const path = lang ? `${apiPath}?lang=${encodeURIComponent(lang)}` : apiPath
  const res = await api.get<LegalDocumentResponse>(path)
  return res.data
}

export function LegalReadingDialog(props: LegalReadingDialogProps) {
  const { t, i18n } = useTranslation()
  const { open, onOpenChange, doc } = props
  const lang = i18n.resolvedLanguage ?? i18n.language ?? ''

  const displayName = doc.name || t(doc.nameKey)

  const { data, isLoading } = useQuery({
    queryKey: ['legal-reading-doc', doc.apiPath, lang],
    queryFn: () => fetchLegalDocument(doc.apiPath, lang),
    enabled: open,
    staleTime: 10 * 60 * 1000,
  })

  const rawContent = data?.data?.trim() ?? ''
  const isUrl = rawContent.length > 0 && isHttpUrl(rawContent)
  const contentIsHtml = rawContent.length > 0 && isLikelyHtml(rawContent)

  const renderBody = () => {
    if (isLoading) {
      return (
        <div className='space-y-3'>
          <Skeleton className='h-6 w-[40%]' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-[90%]' />
          <Skeleton className='h-4 w-[95%]' />
        </div>
      )
    }
    if (isUrl) {
      return (
        <div className='space-y-3'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'The administrator configured an external link for this document.'
            )}
          </p>
          <Button
            render={
              <a href={rawContent} target='_blank' rel='noopener noreferrer' />
            }
          >
            {t('View document')}
          </Button>
        </div>
      )
    }
    if (rawContent.length > 0) {
      return (
        <RichContent
          mode={contentIsHtml ? 'html' : 'markdown'}
          content={rawContent}
          className='prose-neutral dark:prose-invert max-w-none'
        />
      )
    }
    return (
      <div className='text-muted-foreground flex items-center gap-2 text-sm'>
        <FileWarning className='h-4 w-4' />
        {data?.message || t('This document is currently unavailable.')}
      </div>
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={displayName}
      contentClassName='max-w-2xl'
      headerClassName='text-left'
      contentHeight='65vh'
      bodyClassName='flex h-full flex-col gap-3'
      footer={
        <Button
          type='button'
          variant='outline'
          onClick={() => onOpenChange(false)}
        >
          {t('Close')}
        </Button>
      }
    >
      <div className='min-h-0 flex-1 overflow-y-auto pr-1'>
        <div className='rounded-md border px-3 py-2 sm:px-4 sm:py-3'>
          {renderBody()}
        </div>
      </div>
    </Dialog>
  )
}
