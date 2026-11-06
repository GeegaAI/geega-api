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

import { getErrorReasonLabel } from '../constants'
import { formatErrorCount } from '../lib/format'
import type { ErrorReasonStat } from '../types'
import {
  ErrorSampleMessage,
  ErrorShareBar,
  ErrorStatusCodes,
} from './error-table-cells'

interface ErrorReasonTableProps {
  rows: ErrorReasonStat[]
}

export function ErrorReasonTable(props: ErrorReasonTableProps) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t('Reason Distribution')}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="max-h-[480px] overflow-auto [&_[data-slot=table-container]]:overflow-visible [&_table]:border-separate [&_table]:border-spacing-0 [&_td]:border-r [&_td]:border-b [&_td:first-child]:border-l [&_th]:border-r [&_th]:border-b [&_thead_tr:first-child_th]:border-t [&_thead_tr:first-child_th:first-child]:border-l">
          <Table className="min-w-[760px]">
            <TableHeader className="sticky top-0 z-10 bg-(--table-header) [&_th]:text-center">
              <TableRow>
                <TableHead className="text-left">{t('Reason')}</TableHead>
                <TableHead>{t('Error Count')}</TableHead>
                <TableHead>{t('Share of Errors')}</TableHead>
                <TableHead>{t('Status Code')}</TableHead>
                <TableHead>{t('Affected Channels')}</TableHead>
                <TableHead className="text-left">
                  {t('Sample Message')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {props.rows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="text-muted-foreground h-24 text-center"
                  >
                    {t('No data')}
                  </TableCell>
                </TableRow>
              ) : (
                props.rows.map((row) => (
                  <TableRow key={row.code}>
                    <TableCell className="font-medium">
                      {getErrorReasonLabel(row.code, t)}
                    </TableCell>
                    <TableCell className="text-center font-mono">
                      {formatErrorCount(row.count)}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-center">
                        <ErrorShareBar ratio={row.ratio} />
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-center">
                        <ErrorStatusCodes codes={row.status_codes} />
                      </div>
                    </TableCell>
                    <TableCell className="text-center font-mono">
                      {formatErrorCount(row.channel_count)}
                    </TableCell>
                    <TableCell>
                      <ErrorSampleMessage message={row.sample_message} />
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
