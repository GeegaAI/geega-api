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
import { useTranslation } from 'react-i18next'

import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'

import {
  formatErrorCount,
  formatErrorRate,
  getErrorRateTextClass,
} from '../lib/format'
import type { ErrorAnalysisSummary } from '../types'

interface ErrorSummaryCardsProps {
  summary?: ErrorAnalysisSummary
}

export function ErrorSummaryCards(props: ErrorSummaryCardsProps) {
  const { t } = useTranslation()
  const summary = props.summary

  const cards: {
    label: string
    value: string
    hint?: string
    className?: string
  }[] = [
    {
      label: t('Error Count'),
      value: summary ? formatErrorCount(summary.error_count) : '—',
    },
    {
      label: t('Error Rate'),
      value: summary ? formatErrorRate(summary.error_rate) : '—',
      hint: t('Errors / total requests'),
      className: summary ? getErrorRateTextClass(summary.error_rate) : '',
    },
    {
      label: t('Affected Users'),
      value: summary ? formatErrorCount(summary.affected_users) : '—',
    },
    {
      label: t('Affected Channels'),
      value: summary ? formatErrorCount(summary.affected_channels) : '—',
    },
    {
      label: t('Affected Models'),
      value: summary ? formatErrorCount(summary.affected_models) : '—',
    },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
      {cards.map((card) => (
        <Card key={card.label} className="h-full py-4">
          <CardContent className="space-y-1 px-4">
            <div className="text-muted-foreground text-xs">{card.label}</div>
            <div
              className={cn(
                'font-mono text-lg font-semibold',
                card.className
              )}
            >
              {card.value}
            </div>
            {card.hint ? (
              <div className="text-muted-foreground text-[11px]">
                {card.hint}
              </div>
            ) : null}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
