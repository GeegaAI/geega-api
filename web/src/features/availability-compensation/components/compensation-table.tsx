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
import { ChevronRight } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'

import {
  compensationFailCount,
  groupCompensationRowsBySupplier,
  type CompensationRowGroup,
} from '../lib/derive'
import {
  formatAvailabilityCompensationCount,
  formatAvailabilityCompensationPercent,
  hasCompensation,
} from '../lib/format'
import type {
  AvailabilityCompensationRow,
  AvailabilityCompensationSupplierRow,
  AvailabilityCompensationVersion,
} from '../types'
import { CompensationRatioCell, CompensationTierCell } from './compensation-cells'

interface CompensationTableProps {
  rows: AvailabilityCompensationRow[]
  suppliers: AvailabilityCompensationSupplierRow[]
  version: AvailabilityCompensationVersion
}

export function CompensationTable(props: CompensationTableProps) {
  const { t } = useTranslation()
  const version = props.version
  const groups = useMemo(
    () => groupCompensationRowsBySupplier(props.rows, props.suppliers, version),
    [props.rows, props.suppliers, version]
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          {t('Compensation Detail')}
          <span className="text-muted-foreground text-xs font-normal">
            {version === 'all'
              ? t('All Errors')
              : t('Supplier Errors Only')}
          </span>
        </CardTitle>
        <p className="text-muted-foreground text-xs">
          {t('Grouped by model, channel and supplier.')}
        </p>
        <p className="text-muted-foreground text-xs">
          {t(
            'Supplier results aggregate all model × channel minute rows, so the same minute can be counted more than once.'
          )}
        </p>
      </CardHeader>
      <CardContent>
        <div className="max-h-[640px] overflow-auto [&_[data-slot=table-container]]:overflow-visible [&_table]:border-separate [&_table]:border-spacing-0 [&_td]:border-r [&_td]:border-b [&_td:first-child]:border-l [&_th]:border-r [&_th]:border-b [&_thead_tr:first-child_th]:border-t [&_thead_tr:first-child_th:first-child]:border-l">
          <Table className="min-w-[1360px]">
            <TableHeader className="sticky top-0 z-10 bg-(--table-header) [&_th]:text-center">
              <TableRow>
                <TableHead className="text-left">{t('Supplier / Model')}</TableHead>
                <TableHead>{t('Channel ID')}</TableHead>
                <TableHead className="text-left">{t('Channel Name')}</TableHead>
                <TableHead>{t('Successful Requests')}</TableHead>
                <TableHead>
                  {version === 'all'
                    ? t('Failed Requests (All)')
                    : t('Service-side Errors')}
                </TableHead>
                <TableHead>{t('Error Rate')}</TableHead>
                <TableHead>{t('Error Minutes (≥50%)')}</TableHead>
                <TableHead>{t('Error Minutes (Any)')}</TableHead>
                <TableHead>{t('Availability (≥50%)')}</TableHead>
                <TableHead>{t('Availability (Any)')}</TableHead>
                <TableHead>{t('Applicable Tier')}</TableHead>
                <TableHead>{t('Compensation Ratio')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={12}
                    className="text-muted-foreground h-24 text-center"
                  >
                    {t('No data')}
                  </TableCell>
                </TableRow>
              ) : (
                groups.map((group) => (
                  <SupplierGroup
                    key={group.supplier.supplier}
                    group={group}
                    version={version}
                  />
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

function SupplierGroup(props: {
  group: CompensationRowGroup
  version: AvailabilityCompensationVersion
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const version = props.version
  const supplier = props.group.supplier
  const tier = version === 'all' ? supplier.tier_all : supplier.tier_service
  const fail = compensationFailCount(supplier, version)
  const errorRate =
    version === 'all' ? supplier.error_rate_all : supplier.error_rate_service
  const errorMinutesGe50 =
    version === 'all'
      ? supplier.error_minutes_ge50_all
      : supplier.error_minutes_ge50_service
  const errorMinutesAny =
    version === 'all'
      ? supplier.error_minutes_any_all
      : supplier.error_minutes_any_service
  const availabilityGe50 =
    version === 'all'
      ? supplier.availability_ge50_all
      : supplier.availability_ge50_service
  const availabilityAny =
    version === 'all'
      ? supplier.availability_any_all
      : supplier.availability_any_service
  const ratio =
    version === 'all'
      ? supplier.compensation_pct_all
      : supplier.compensation_pct_service

  return (
    <>
      <TableRow className={cn(hasCompensation(tier) ? 'bg-red-500/5' : 'bg-muted/40')}>
        <TableCell colSpan={3} className="font-semibold">
          <button
            type="button"
            className="-mx-1 flex items-center gap-1 rounded px-1 transition-colors hover:bg-muted/60"
            aria-expanded={expanded}
            onClick={() => setExpanded(!expanded)}
          >
            <ChevronRight
              aria-hidden="true"
              className={cn(
                'size-4 shrink-0 transition-transform',
                expanded ? 'rotate-90' : ''
              )}
            />
            {supplier.supplier}
          </button>
          <span className="text-muted-foreground ms-2 font-normal">
            · {t('Channels')} {formatAvailabilityCompensationCount(supplier.channel_count)}
          </span>
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationCount(supplier.ok_requests)}
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationCount(fail)}
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationPercent(errorRate)}
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationCount(errorMinutesGe50)}
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationCount(errorMinutesAny)}
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationPercent(availabilityGe50)}
        </TableCell>
        <TableCell className="text-center font-mono">
          {formatAvailabilityCompensationPercent(availabilityAny)}
        </TableCell>
        <TableCell className="text-center">
          <CompensationTierCell tier={tier} />
        </TableCell>
        <TableCell className="text-center">
          <CompensationRatioCell ratio={ratio} />
        </TableCell>
      </TableRow>
      {expanded
        ? props.group.rows.map((row) => (
            <DetailRow
              key={`${row.channel_id}-${row.model_name}`}
              row={row}
              version={version}
            />
          ))
        : null}
    </>
  )
}

function DetailRow(props: {
  row: AvailabilityCompensationRow
  version: AvailabilityCompensationVersion
}) {
  const version = props.version
  const row = props.row
  const tier = version === 'all' ? row.tier_all : row.tier_service
  const fail = compensationFailCount(row, version)
  const errorRate =
    version === 'all' ? row.error_rate_all : row.error_rate_service
  const errorMinutesGe50 =
    version === 'all'
      ? row.error_minutes_ge50_all
      : row.error_minutes_ge50_service
  const errorMinutesAny =
    version === 'all' ? row.error_minutes_any_all : row.error_minutes_any_service
  const availabilityGe50 =
    version === 'all'
      ? row.availability_ge50_all
      : row.availability_ge50_service
  const availabilityAny =
    version === 'all' ? row.availability_any_all : row.availability_any_service
  const ratio =
    version === 'all' ? row.compensation_pct_all : row.compensation_pct_service

  return (
    <TableRow className={hasCompensation(tier) ? 'bg-red-500/5' : ''}>
      <TableCell className="ps-6 font-mono text-xs">
        {row.model_name || '—'}
      </TableCell>
      <TableCell className="text-center font-mono">{row.channel_id}</TableCell>
      <TableCell>{row.channel_name}</TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationCount(row.ok_requests)}
      </TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationCount(fail)}
      </TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationPercent(errorRate)}
      </TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationCount(errorMinutesGe50)}
      </TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationCount(errorMinutesAny)}
      </TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationPercent(availabilityGe50)}
      </TableCell>
      <TableCell className="text-center font-mono">
        {formatAvailabilityCompensationPercent(availabilityAny)}
      </TableCell>
      <TableCell className="text-center">
        <CompensationTierCell tier={tier} />
      </TableCell>
      <TableCell className="text-center">
        <CompensationRatioCell ratio={ratio} />
      </TableCell>
    </TableRow>
  )
}
