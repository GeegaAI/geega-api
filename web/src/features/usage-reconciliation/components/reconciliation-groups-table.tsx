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

import { Badge } from '@/components/ui/badge'
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

import { getUsageBillingPathLabelKey } from '../constants'
import {
  computeRowInputDelta,
  deltaColorClass,
  formatDelta,
  formatTokenCount,
} from '../lib/derive'
import type { UsageReconciliationRow } from '../types'

interface ReconciliationGroupsTableProps {
  groups: UsageReconciliationRow[]
}

export function ReconciliationGroupsTable(props: ReconciliationGroupsTableProps) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t('Usage Source')}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="max-h-[560px] overflow-auto [&_[data-slot=table-container]]:overflow-visible [&_table]:border-separate [&_table]:border-spacing-0 [&_td]:border-r [&_td]:border-b [&_td:first-child]:border-l [&_th]:border-r [&_th]:border-b [&_thead_tr:first-child_th]:border-t [&_thead_tr:first-child_th:first-child]:border-l">
          <Table className="min-w-[820px]">
            <TableHeader className="sticky top-0 z-10 bg-(--table-header) [&_th]:text-center">
              <TableRow>
                <TableHead rowSpan={2}>{t('Usage Source')}</TableHead>
                <TableHead rowSpan={2}>
                  {t('Request Count')}
                </TableHead>
                <TableHead colSpan={3}>
                  {t('Input Tokens')}
                </TableHead>
                <TableHead rowSpan={2}>
                  {t('Output Tokens')}
                </TableHead>
              </TableRow>
              <TableRow>
                <TableHead>{t('Upstream')}</TableHead>
                <TableHead>{t('System')}</TableHead>
                <TableHead>{t('Difference')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {props.groups.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-muted-foreground h-24 text-center">
                    {t('No data')}
                  </TableCell>
                </TableRow>
              ) : (
                props.groups.map((row) => (
                  <TableRow
                    key={`${row.usage_billing_path}-${row.semantic}`}
                    className={
                      row.estimated || row.local
                        ? 'bg-amber-500/5'
                        : undefined
                    }
                  >
                    <TableCell>
                      <span title={row.usage_billing_path}>
                        {t(getUsageBillingPathLabelKey(row.usage_billing_path))}
                      </span>
                      {row.estimated ? (
                        <Badge variant="outline" className="ml-2 border-amber-500 text-amber-600">
                          {t('Estimated')}
                        </Badge>
                      ) : null}
                      {row.usage_billing_path === 'legacy' ? (
                        <Badge variant="outline" className="ml-2">
                          {t('Legacy')}
                        </Badge>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono">
                      {formatTokenCount(row.request_count)}
                    </TableCell>
                    {row.reconcilable ? (
                      <>
                        <TableCell className="font-mono">
                          {formatTokenCount(row.upstream_input_tokens)}
                        </TableCell>
                        <TableCell className="font-mono">
                          {formatTokenCount(row.system_input_tokens)}
                        </TableCell>
                        <DeltaCell value={computeRowInputDelta(row)} />
                      </>
                    ) : (
                      <>
                        <UpstreamPendingCell />
                        <TableCell className="font-mono">
                          {formatTokenCount(row.system_input_tokens)}
                        </TableCell>
                        <UpstreamPendingCell />
                      </>
                    )}
                    <TableCell className="font-mono">
                      {formatTokenCount(row.system_output_tokens)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

function DeltaCell(props: { value: number }) {
  const value = props.value
  return (
    <TableCell className={`font-mono ${deltaColorClass(value)}`}>
      {formatDelta(value)}
    </TableCell>
  )
}

function UpstreamPendingCell() {
  return <TableCell className="text-muted-foreground font-mono">—</TableCell>
}
