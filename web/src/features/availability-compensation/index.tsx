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
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Card, CardContent } from '@/components/ui/card'

import {
  downloadAvailabilityCompensation,
  getAvailabilityCompensation,
} from './api'
import { CompensationFilter } from './components/compensation-filter'
import { CompensationGuide } from './components/compensation-guide'
import { CompensationSummaryCards } from './components/compensation-summary-cards'
import { CompensationTable } from './components/compensation-table'
import {
  formatAvailabilityCompensationCount,
  formatAvailabilityCompensationDateTime,
} from './lib/format'
import {
  formatAvailabilityCompensationDates,
  isAvailabilityCompensationRangeInvalid,
  resolveAvailabilityCompensationRange,
  type AvailabilityCompensationRangePreset,
} from './lib/range'
import type { AvailabilityCompensationVersion } from './types'

interface CommittedQuery {
  preset: AvailabilityCompensationRangePreset
  startDate: string
  endDate: string
}

function currentMonthQuery(): CommittedQuery {
  const now = dayjs()
  return {
    preset: 'thisMonth',
    startDate: now.startOf('month').format('YYYY-MM-DD'),
    endDate: now.format('YYYY-MM-DD'),
  }
}

export function AvailabilityCompensation() {
  const { t } = useTranslation()
  const [preset, setPreset] = useState<AvailabilityCompensationRangePreset>(
    'thisMonth'
  )
  const [startDate, setStartDate] = useState(
    () => currentMonthQuery().startDate
  )
  const [endDate, setEndDate] = useState(() => currentMonthQuery().endDate)
  const [query, setQuery] = useState<CommittedQuery>(currentMonthQuery)
  const [isExporting, setIsExporting] = useState(false)
  const [version, setVersion] = useState<AvailabilityCompensationVersion>('all')

  const range = useMemo(
    () => resolveAvailabilityCompensationRange(query.preset, query.startDate, query.endDate),
    [query]
  )
  const queryDirty =
    query.preset !== preset ||
    query.startDate !== startDate ||
    query.endDate !== endDate
  const invalidRange = isAvailabilityCompensationRangeInvalid(startDate, endDate)

  const { data, isFetching, isError } = useQuery({
    queryKey: [
      'availability-compensation',
      range.startTimestamp,
      range.endTimestamp,
    ],
    queryFn: () =>
      getAvailabilityCompensation({
        startTimestamp: range.startTimestamp,
        endTimestamp: range.endTimestamp,
      }),
    placeholderData: keepPreviousData,
  })

  const result = data?.data

  const handlePresetChange = (value: AvailabilityCompensationRangePreset) => {
    const now = dayjs()
    let monthStart: dayjs.Dayjs
    let inputEnd: dayjs.Dayjs
    if (value === 'lastMonth') {
      const lastMonth = now.subtract(1, 'month')
      monthStart = lastMonth.startOf('month')
      inputEnd = lastMonth.endOf('month')
    } else {
      monthStart = now.startOf('month')
      inputEnd = now
    }
    const nextStartDate = monthStart.format('YYYY-MM-DD')
    const nextEndDate = inputEnd.format('YYYY-MM-DD')
    setPreset(value)
    setStartDate(nextStartDate)
    setEndDate(nextEndDate)
    setQuery({ preset: value, startDate: nextStartDate, endDate: nextEndDate })
  }

  const handleSearch = () => {
    setQuery({ preset, startDate, endDate })
  }

  const handleExport = async () => {
    setIsExporting(true)
    try {
      const dates = formatAvailabilityCompensationDates(range)
      await downloadAvailabilityCompensation(
        {
          startTimestamp: range.startTimestamp,
          endTimestamp: range.endTimestamp,
        },
        `供应商可用性赔偿测算-${dates.startDate}-${dates.endDate}.xlsx`
      )
      toast.success(t('Excel exported'))
    } catch {
      toast.error(t('Failed to export Excel'))
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Supplier Availability Compensation')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-4">
          <p className="text-muted-foreground text-sm">
            {t(
              'Measure supplier availability from request logs by minute and calculate contractual compensation ratios.'
            )}
          </p>

          <CompensationFilter
            preset={preset}
            startDate={startDate}
            endDate={endDate}
            version={version}
            queryDirty={queryDirty}
            isFetching={isFetching}
            isExporting={isExporting}
            onPresetChange={handlePresetChange}
            onVersionChange={setVersion}
            onStartDateChange={(value) => {
              setStartDate(value)
              setPreset('custom')
            }}
            onEndDateChange={(value) => {
              setEndDate(value)
              setPreset('custom')
            }}
            onSearch={handleSearch}
            onExport={() => void handleExport()}
          />

          {isError ? (
            <p className="text-destructive text-sm">{t('Failed to load')}</p>
          ) : null}

          {result ? (
            <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
              <span>
                {t('Data coverage: {{start}} – {{end}}', {
                  start: formatAvailabilityCompensationDateTime(
                    result.data_start_timestamp
                  ),
                  end: formatAvailabilityCompensationDateTime(
                    result.data_end_timestamp
                  ),
                })}
              </span>
              <span>
                {t('Period covers {{minutes}} minutes.', {
                  minutes: formatAvailabilityCompensationCount(
                    result.total_minutes
                  ),
                })}
              </span>
              {result.snapshot ? (
                <span className="text-amber-600 dark:text-amber-400">
                  {t(
                    "The selected period has not ended yet; availability is calculated against the full period's minutes."
                  )}
                </span>
              ) : null}
            </div>
          ) : null}

          <CompensationSummaryCards data={result} version={version} />

          {result && result.suppliers.length === 0 && !invalidRange ? (
            <Card>
              <CardContent className="text-muted-foreground py-8 text-center text-sm">
                {t('No requests in the selected range.')}
              </CardContent>
            </Card>
          ) : (
            <CompensationTable
              rows={result?.rows ?? []}
              suppliers={result?.suppliers ?? []}
              version={version}
            />
          )}

          <CompensationGuide />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
