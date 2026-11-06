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
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

import { getUsageReconciliation } from './api'
import { ReconciliationChannelTable } from './components/reconciliation-channel-table'
import { ReconciliationGroupsTable } from './components/reconciliation-groups-table'
import { ReconciliationGuide } from './components/reconciliation-guide'
import { ReconciliationSummaryCards } from './components/reconciliation-summary-cards'
import { aggregateReconciliationRowsByPath } from './lib/derive'
import {
  resolveReconciliationRange,
  type ReconciliationRangePreset,
} from './lib/range'

const RANGE_PRESETS: { value: ReconciliationRangePreset; labelKey: string }[] = [
  { value: 'today', labelKey: 'Today' },
  { value: 'last7days', labelKey: 'Last 7 Days' },
  { value: 'last30days', labelKey: 'Last 30 Days' },
]

interface CommittedQuery {
  preset: ReconciliationRangePreset
  startDate: string
  endDate: string
}

export function UsageReconciliation() {
  const { t } = useTranslation()
  const [preset, setPreset] = useState<ReconciliationRangePreset>('last7days')
  const [startDate, setStartDate] = useState(
    dayjs().subtract(6, 'day').format('YYYY-MM-DD')
  )
  const [endDate, setEndDate] = useState(dayjs().format('YYYY-MM-DD'))
  const [query, setQuery] = useState<CommittedQuery>(() => ({
    preset,
    startDate,
    endDate,
  }))

  const range = useMemo(
    () => resolveReconciliationRange(query.preset, query.startDate, query.endDate),
    [query]
  )
  const queryDirty =
    query.preset !== preset ||
    query.startDate !== startDate ||
    query.endDate !== endDate
  const { data, isFetching, isError } = useQuery({
    queryKey: ['usage-reconciliation', range.startTimestamp, range.endTimestamp],
    queryFn: () => getUsageReconciliation(range.startTimestamp, range.endTimestamp),
    placeholderData: keepPreviousData,
  })

  const groups = useMemo(() => data?.data?.groups ?? [], [data])
  const pathGroups = useMemo(
    () => aggregateReconciliationRowsByPath(groups),
    [groups]
  )

  const selectPreset = (value: ReconciliationRangePreset) => {
    setPreset(value)
    const today = dayjs().format('YYYY-MM-DD')
    let start = startDate
    let end = endDate
    if (value === 'today') {
      start = today
      end = today
    } else if (value === 'last7days') {
      start = dayjs().subtract(6, 'day').format('YYYY-MM-DD')
      end = today
    } else if (value === 'last30days') {
      start = dayjs().subtract(29, 'day').format('YYYY-MM-DD')
      end = today
    }
    setStartDate(start)
    setEndDate(end)
    setQuery({ preset: value, startDate: start, endDate: end })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Token Reconciliation')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-4">
          <p className="text-muted-foreground text-sm">
            {t('Reconcile system token usage against upstream billing data')}
          </p>

          <Card className="py-4">
            <CardContent className="flex flex-wrap items-center gap-3 px-4">
              {RANGE_PRESETS.map((item) => (
                <Button
                  key={item.value}
                  size="sm"
                  variant={preset === item.value ? 'default' : 'outline'}
                  onClick={() => selectPreset(item.value)}
                >
                  {t(item.labelKey)}
                </Button>
              ))}
              <div className="flex items-center gap-2">
                <Input
                  type="date"
                  className="w-40"
                  value={startDate}
                  onChange={(event) => {
                    setStartDate(event.target.value)
                    setPreset('custom')
                  }}
                />
                <span className="text-muted-foreground text-sm">~</span>
                <Input
                  type="date"
                  className="w-40"
                  value={endDate}
                  onChange={(event) => {
                    setEndDate(event.target.value)
                    setPreset('custom')
                  }}
                />
              </div>
              <Button
                size="sm"
                disabled={!queryDirty || isFetching}
                onClick={() => setQuery({ preset, startDate, endDate })}
              >
                {isFetching ? <Spinner /> : null}
                {t('Search')}
              </Button>
            </CardContent>
          </Card>

          {isError ? (
            <p className="text-destructive text-sm">{t('Failed to load')}</p>
          ) : null}

          <ReconciliationSummaryCards groups={groups} />
          <ReconciliationChannelTable rows={groups} />
          <ReconciliationGroupsTable groups={pathGroups} />
          <ReconciliationGuide />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
