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

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

import { ERROR_ANALYSIS_RANGE_PRESETS } from '../constants'
import type { ErrorAnalysisRangePreset } from '../lib/range'

interface ErrorAnalysisFilterProps {
  preset: ErrorAnalysisRangePreset
  startDate: string
  endDate: string
  channelId: string
  modelName: string
  group: string
  isFetching: boolean
  onPresetChange: (preset: ErrorAnalysisRangePreset) => void
  onStartDateChange: (value: string) => void
  onEndDateChange: (value: string) => void
  onChannelIdChange: (value: string) => void
  onModelNameChange: (value: string) => void
  onGroupChange: (value: string) => void
  onSearch: () => void
}

export function ErrorAnalysisFilter(props: ErrorAnalysisFilterProps) {
  const { t } = useTranslation()
  const invalidRange =
    props.preset === 'custom' && props.startDate > props.endDate

  return (
    <Card className="py-4">
      <CardContent className="flex flex-col gap-3 px-4">
        <div className="flex flex-wrap items-center gap-2">
          {ERROR_ANALYSIS_RANGE_PRESETS.map((item) => (
            <Button
              key={item.value}
              size="sm"
              variant={props.preset === item.value ? 'default' : 'outline'}
              onClick={() => props.onPresetChange(item.value)}
            >
              {t(item.labelKey)}
            </Button>
          ))}
          <div className="flex items-center gap-2">
            <Input
              type="date"
              className="w-40"
              aria-label={t('Start Date')}
              value={props.startDate}
              onChange={(event) => props.onStartDateChange(event.target.value)}
            />
            <span className="text-muted-foreground text-sm">~</span>
            <Input
              type="date"
              className="w-40"
              aria-label={t('End Date')}
              value={props.endDate}
              onChange={(event) => props.onEndDateChange(event.target.value)}
            />
          </div>
          <Input
            className="w-36"
            inputMode="numeric"
            aria-label={t('Channel ID')}
            placeholder={t('Channel ID')}
            value={props.channelId}
            onChange={(event) =>
              props.onChannelIdChange(event.target.value.replaceAll(/\D/g, ''))
            }
          />
          <Input
            className="w-44"
            aria-label={t('Model')}
            placeholder={t('Model')}
            value={props.modelName}
            onChange={(event) => props.onModelNameChange(event.target.value)}
          />
          <Input
            className="w-32"
            aria-label={t('Group')}
            placeholder={t('Group')}
            value={props.group}
            onChange={(event) => props.onGroupChange(event.target.value)}
          />
          <Button
            size="sm"
            disabled={props.isFetching || invalidRange}
            onClick={props.onSearch}
          >
            {props.isFetching ? <Spinner /> : null}
            {t('Search')}
          </Button>
        </div>
        <p className="text-muted-foreground text-xs">
          {t('Error rate = errors / (successful + failed requests).')}
        </p>
        {invalidRange ? (
          <p className="text-destructive text-xs">
            {t('Start date must not be later than end date.')}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
