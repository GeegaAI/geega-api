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
import { VChart } from '@visactor/react-vchart'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import {
  formatTrendBucket,
  formatTrendTooltip,
} from '../lib/format'
import type { ErrorTrendPoint } from '../types'

interface ErrorTrendChartProps {
  trend: ErrorTrendPoint[]
  bucket: 'hour' | 'day'
}

export function ErrorTrendChart(props: ErrorTrendChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const textColor =
    resolvedTheme === 'dark'
      ? 'rgba(255, 255, 255, 0.68)'
      : 'rgba(15, 23, 42, 0.58)'
  const gridColor =
    resolvedTheme === 'dark'
      ? 'rgba(255, 255, 255, 0.12)'
      : 'rgba(15, 23, 42, 0.12)'

  const values = useMemo(
    () =>
      props.trend.map((point) => ({
        time: formatTrendBucket(point.timestamp, props.bucket),
        timestamp: point.timestamp,
        errors: point.error_count,
      })),
    [props.bucket, props.trend]
  )

  const spec = useMemo(() => {
    if (values.length === 0) return null
    return {
      type: 'bar' as const,
      data: [{ id: 'error-trend', values }],
      xField: 'time',
      yField: 'errors',
      bar: {
        style: {
          fill: '#ef4444',
          cornerRadius: 2,
        },
      },
      axes: [
        {
          orient: 'bottom' as const,
          label: { style: { fill: textColor, fontSize: 10 } },
          tick: { visible: false },
        },
        {
          orient: 'left' as const,
          label: { style: { fill: textColor, fontSize: 10 } },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: gridColor },
          },
        },
      ],
      tooltip: {
        mark: {
          title: {
            value: (datum: { timestamp: number }) =>
              formatTrendTooltip(datum.timestamp, props.bucket),
          },
          content: [
            {
              key: t('Error Count'),
              value: (datum: { errors: number }) => datum.errors,
            },
          ],
        },
      },
    }
  }, [gridColor, props.bucket, t, textColor, values])

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <CardTitle className="text-base">{t('Error Trend')}</CardTitle>
        <span className="text-muted-foreground text-xs">
          {t(props.bucket === 'hour' ? 'Hourly' : 'Daily')}
        </span>
      </CardHeader>
      <CardContent>
        {values.length === 0 ? (
          <div className="text-muted-foreground flex h-64 items-center justify-center rounded-lg border text-sm">
            {t('No data')}
          </div>
        ) : (
          <div className="h-64">
            {themeReady && spec ? (
              <VChart
                key={`error-trend-${resolvedTheme}`}
                spec={{
                  ...spec,
                  theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                  background: 'transparent',
                }}
                option={VCHART_OPTION}
              />
            ) : null}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
