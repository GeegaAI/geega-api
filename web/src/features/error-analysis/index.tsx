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
import { CircleCheck } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Card, CardContent } from '@/components/ui/card'

import { getErrorAnalysis } from './api'
import { ErrorAnalysisFilter } from './components/error-analysis-filter'
import { ErrorDetailTable } from './components/error-detail-table'
import { ErrorReasonTable } from './components/error-reason-table'
import { ErrorSummaryCards } from './components/error-summary-cards'
import { ErrorTrendChart } from './components/error-trend-chart'
import {
  resolveErrorAnalysisRange,
  type ErrorAnalysisRangePreset,
} from './lib/range'

interface CommittedErrorAnalysisQuery {
  preset: ErrorAnalysisRangePreset
  startDate: string
  endDate: string
  channelId: string
  modelName: string
  group: string
}

export function ErrorAnalysis() {
  const { t } = useTranslation()
  const [preset, setPreset] = useState<ErrorAnalysisRangePreset>('today')
  const [startDate, setStartDate] = useState(() =>
    dayjs().format('YYYY-MM-DD')
  )
  const [endDate, setEndDate] = useState(() => dayjs().format('YYYY-MM-DD'))
  const [channelId, setChannelId] = useState('')
  const [modelName, setModelName] = useState('')
  const [group, setGroup] = useState('')
  const [query, setQuery] = useState<CommittedErrorAnalysisQuery>(() => {
    const today = dayjs().format('YYYY-MM-DD')
    return {
      preset: 'today',
      startDate: today,
      endDate: today,
      channelId: '',
      modelName: '',
      group: '',
    }
  })

  const range = useMemo(
    () => resolveErrorAnalysisRange(query.preset, query.startDate, query.endDate),
    [query]
  )

  const { data, isFetching, isError } = useQuery({
    queryKey: [
      'error-analysis',
      range.startTimestamp,
      range.endTimestamp,
      query.channelId,
      query.modelName,
      query.group,
    ],
    queryFn: () =>
      getErrorAnalysis({
        startTimestamp: range.startTimestamp,
        endTimestamp: range.endTimestamp,
        channelId: query.channelId ? Number(query.channelId) : undefined,
        modelName: query.modelName.trim() || undefined,
        group: query.group.trim() || undefined,
      }),
    placeholderData: keepPreviousData,
  })

  const result = data?.data

  const handlePresetChange = (value: ErrorAnalysisRangePreset) => {
    const now = dayjs()
    let start = now
    let end = now
    if (value === 'yesterday') {
      start = now.subtract(1, 'day')
      end = start
    } else if (value === 'last7days') {
      start = now.subtract(6, 'day')
      end = now
    }
    const nextStartDate = start.format('YYYY-MM-DD')
    const nextEndDate = end.format('YYYY-MM-DD')
    setPreset(value)
    setStartDate(nextStartDate)
    setEndDate(nextEndDate)
    setQuery({
      preset: value,
      startDate: nextStartDate,
      endDate: nextEndDate,
      channelId,
      modelName,
      group,
    })
  }

  const handleSearch = () => {
    setQuery({
      preset,
      startDate,
      endDate,
      channelId,
      modelName,
      group,
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Error Analysis')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-4">
          <p className="text-muted-foreground text-sm">
            {t(
              'Inspect upstream errors by channel, model and reason for daily operations.'
            )}
          </p>

          <ErrorAnalysisFilter
            preset={preset}
            startDate={startDate}
            endDate={endDate}
            channelId={channelId}
            modelName={modelName}
            group={group}
            isFetching={isFetching}
            onPresetChange={handlePresetChange}
            onStartDateChange={(value) => {
              setStartDate(value)
              setPreset('custom')
            }}
            onEndDateChange={(value) => {
              setEndDate(value)
              setPreset('custom')
            }}
            onChannelIdChange={setChannelId}
            onModelNameChange={setModelName}
            onGroupChange={setGroup}
            onSearch={handleSearch}
          />

          {isError ? (
            <p className="text-destructive text-sm">{t('Failed to load')}</p>
          ) : null}
          {result?.truncated ? (
            <p className="text-amber-600 text-xs dark:text-amber-400">
              {t(
                'Only the latest 50,000 error logs are included in the detail statistics.'
              )}
            </p>
          ) : null}

          <ErrorSummaryCards summary={result?.summary} />

          {result && result.summary.error_count === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center gap-2 py-12">
                <CircleCheck
                  className="size-8 text-emerald-600 dark:text-emerald-400"
                  aria-hidden="true"
                />
                <p className="text-sm font-medium">
                  {t('No upstream errors in the selected range.')}
                </p>
              </CardContent>
            </Card>
          ) : (
            <>
              <ErrorTrendChart
                trend={result?.trend ?? []}
                bucket={result?.bucket ?? 'hour'}
              />
              <ErrorReasonTable rows={result?.reasons ?? []} />
              <ErrorDetailTable
                rows={result?.channels ?? []}
                breakdownRows={result?.rows ?? []}
                rowsTruncated={result?.rows_truncated ?? false}
                range={range}
              />
            </>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
