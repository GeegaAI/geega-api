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

import {
  computeReconcilableShare,
  deltaColorClass,
  formatDelta,
  formatPercent,
  formatTokenCount,
  isReconcilable,
} from '../lib/derive'
import type { UsageReconciliationRow } from '../types'

interface ReconciliationSummaryCardsProps {
  groups: UsageReconciliationRow[]
}

export function ReconciliationSummaryCards(props: ReconciliationSummaryCardsProps) {
  const { t } = useTranslation()
  const rows = props.groups
  const reconciled = rows.filter((row) => isReconcilable(row))

  const sum = (source: UsageReconciliationRow[], pick: (row: UsageReconciliationRow) => number) =>
    source.reduce((total, row) => total + pick(row), 0)

  const inputDiff = sum(reconciled, (row) => row.upstream_input_tokens - row.system_input_tokens)

  const cards: { label: string; value: string; className?: string }[] = [
    {
      label: t('Request Count'),
      value: formatTokenCount(sum(rows, (row) => row.request_count)),
    },
    {
      label: t('Upstream Input (Reconciled)'),
      value: formatTokenCount(sum(reconciled, (row) => row.upstream_input_tokens)),
    },
    {
      label: t('System Input Tokens'),
      value: formatTokenCount(sum(rows, (row) => row.system_input_tokens)),
    },
    {
      label: t('Input Difference'),
      value: formatDelta(inputDiff),
      className: deltaColorClass(inputDiff),
    },
    {
      label: t('Reconcilable Share'),
      value: formatPercent(computeReconcilableShare(rows)),
    },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
      {cards.map((card) => (
        <Card key={card.label} className="h-full py-4">
          <CardContent className="space-y-1 px-4">
            <div className="text-muted-foreground text-xs">{card.label}</div>
            <div className={`font-mono text-lg font-semibold ${card.className ?? ''}`}>
              {card.value}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
