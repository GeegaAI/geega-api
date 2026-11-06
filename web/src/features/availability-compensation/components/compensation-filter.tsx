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
import { Download } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'

import {
  AVAILABILITY_COMPENSATION_PRESETS,
  AVAILABILITY_COMPENSATION_VERSIONS,
  isAvailabilityCompensationVersionValue,
} from '../constants'
import {
  isAvailabilityCompensationRangeInvalid,
  type AvailabilityCompensationRangePreset,
} from '../lib/range'
import type { AvailabilityCompensationVersion } from '../types'

interface CompensationFilterProps {
  preset: AvailabilityCompensationRangePreset
  startDate: string
  endDate: string
  version: AvailabilityCompensationVersion
  queryDirty: boolean
  isFetching: boolean
  isExporting: boolean
  onPresetChange: (preset: AvailabilityCompensationRangePreset) => void
  onVersionChange: (version: AvailabilityCompensationVersion) => void
  onStartDateChange: (value: string) => void
  onEndDateChange: (value: string) => void
  onSearch: () => void
  onExport: () => void
}

export function CompensationFilter(props: CompensationFilterProps) {
  const { t } = useTranslation()
  const invalidRange = isAvailabilityCompensationRangeInvalid(
    props.startDate,
    props.endDate
  )
  const versionItems = useMemo(
    () =>
      AVAILABILITY_COMPENSATION_VERSIONS.map((item) => ({
        value: item.value,
        label: t(item.labelKey),
      })),
    [t]
  )
  const versionLabel =
    versionItems.find((item) => item.value === props.version)?.label ?? ''

  return (
    <Card className="py-4">
      <CardContent className="flex flex-col gap-3 px-4">
        <div className="flex flex-wrap items-center gap-2">
          {AVAILABILITY_COMPENSATION_PRESETS.map((item) => (
            <Button
              key={item.value}
              size="sm"
              variant={props.preset === item.value ? 'default' : 'outline'}
              onClick={() => props.onPresetChange(item.value)}
            >
              {t(item.labelKey)}
            </Button>
          ))}
          <span className="text-muted-foreground text-sm">
            {t('Error Scope')}
          </span>
          <Select
            items={versionItems}
            value={props.version}
            onValueChange={(value) => {
              if (value !== null && isAvailabilityCompensationVersionValue(value)) {
                props.onVersionChange(value)
              }
            }}
          >
            <SelectTrigger className="w-44" aria-label={t('Error Scope')}>
              <SelectValue>{versionLabel}</SelectValue>
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {AVAILABILITY_COMPENSATION_VERSIONS.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {t(item.labelKey)}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
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
          <Button
            size="sm"
            disabled={!props.queryDirty || invalidRange || props.isFetching}
            onClick={props.onSearch}
          >
            {props.isFetching ? <Spinner /> : null}
            {t('Search')}
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={invalidRange || props.isExporting}
            onClick={props.onExport}
          >
            {props.isExporting ? <Spinner /> : <Download aria-hidden="true" />}
            {props.isExporting ? t('Exporting...') : t('Export Excel')}
          </Button>
        </div>
        <p className="text-muted-foreground text-xs">
          {t(
            'A minute counts as an error minute when the error rate within that minute reaches 50%.'
          )}
        </p>
        {invalidRange ? (
          <p className="text-destructive text-xs">
            {t('Please select a valid range of at most 62 days.')}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
