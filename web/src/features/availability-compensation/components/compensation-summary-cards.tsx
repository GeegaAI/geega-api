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

import { countCompensatedSuppliers } from '../lib/derive'
import { formatAvailabilityCompensationCount } from '../lib/format'
import type {
  AvailabilityCompensationData,
  AvailabilityCompensationVersion,
} from '../types'

interface CompensationSummaryCardsProps {
  data?: AvailabilityCompensationData
  version: AvailabilityCompensationVersion
}

export function CompensationSummaryCards(props: CompensationSummaryCardsProps) {
  const { t } = useTranslation()
  const data = props.data
  const version = props.version
  const suppliers = data?.suppliers ?? []
  const compensatedSuppliers = countCompensatedSuppliers(suppliers, version)
  const okRequests = suppliers.reduce((total, row) => total + row.ok_requests, 0)
  const serviceErrors = suppliers.reduce(
    (total, row) => total + row.fail_service_requests,
    0
  )

  const cards: {
    label: string
    value: string
    hint?: string
    className?: string
  }[] = [
    {
      label: t('Suppliers'),
      value: data ? formatAvailabilityCompensationCount(data.supplier_count) : '—',
    },
    {
      label: t('Channels'),
      value: data ? formatAvailabilityCompensationCount(data.channel_count) : '—',
    },
    {
      label: t('Models'),
      value: data ? formatAvailabilityCompensationCount(data.model_count) : '—',
    },
    {
      label: t('Successful Requests'),
      value: data ? formatAvailabilityCompensationCount(okRequests) : '—',
    },
    {
      label: t('Service-side Errors'),
      value: data ? formatAvailabilityCompensationCount(serviceErrors) : '—',
    },
    {
      label: t('Suppliers Requiring Compensation'),
      value: data
        ? formatAvailabilityCompensationCount(compensatedSuppliers)
        : '—',
      hint:
        version === 'all'
          ? t('All Errors')
          : t('Supplier Errors Only'),
      className:
        compensatedSuppliers > 0 ? 'text-red-600 dark:text-red-400' : '',
    },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
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
