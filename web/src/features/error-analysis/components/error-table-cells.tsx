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
import { Check, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import { formatErrorRate } from '../lib/format'

export function ErrorStatusCodes(props: { codes: number[] }) {
  if (props.codes.length === 0) {
    return <span className="text-muted-foreground">—</span>
  }
  return (
    <div className="flex flex-wrap gap-1">
      {props.codes.map((code) => (
        <Badge key={code} variant="outline" className="font-mono text-[11px]">
          {code}
        </Badge>
      ))}
    </div>
  )
}

export function ErrorSampleMessage(props: { message: string }) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })

  if (!props.message) {
    return <span className="text-muted-foreground">—</span>
  }
  return (
    <div className="flex max-w-[360px] items-center gap-1">
      <span className="min-w-0 flex-1 truncate" title={props.message}>
        {props.message}
      </span>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 w-6 shrink-0 p-0"
        aria-label={t('Copy to clipboard')}
        title={t('Copy to clipboard')}
        onClick={() => copyToClipboard(props.message)}
      >
        {copiedText === props.message ? (
          <Check className="size-3.5 text-green-600" />
        ) : (
          <Copy className="text-muted-foreground size-3.5" />
        )}
      </Button>
    </div>
  )
}

export function ErrorShareBar(props: { ratio: number }) {
  const percent = Math.min(100, Math.max(0, props.ratio * 100))
  return (
    <div className="flex items-center gap-2">
      <span className="font-mono text-xs">{formatErrorRate(props.ratio)}</span>
      <div
        className="bg-muted h-1 w-16 overflow-hidden rounded-full"
        aria-hidden="true"
      >
        <div
          className="h-full rounded-full bg-red-500"
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  )
}
