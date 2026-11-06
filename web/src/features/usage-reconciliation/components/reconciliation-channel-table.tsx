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
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'
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

import {
  computeRowInputDelta,
  deltaColorClass,
  formatDelta,
  formatTokenCount,
  groupReconciliationRowsByChannel,
} from '../lib/derive'
import type { UsageReconciliationRow } from '../types'

interface ReconciliationChannelTableProps {
  rows: UsageReconciliationRow[]
}

export function ReconciliationChannelTable(props: ReconciliationChannelTableProps) {
  const { t } = useTranslation()
  const [expandedIds, setExpandedIds] = useState<Set<number>>(() => new Set())
  const channelGroups = groupReconciliationRowsByChannel(props.rows)

  const toggle = (channelId: number) => {
    setExpandedIds((previous) => {
      const next = new Set(previous)
      if (next.has(channelId)) {
        next.delete(channelId)
      } else {
        next.add(channelId)
      }
      return next
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t('Supplier')}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="max-h-[560px] overflow-auto [&_[data-slot=table-container]]:overflow-visible [&_table]:border-separate [&_table]:border-spacing-0 [&_td]:border-r [&_td]:border-b [&_td:first-child]:border-l [&_th]:border-r [&_th]:border-b [&_thead_tr:first-child_th]:border-t [&_thead_tr:first-child_th:first-child]:border-l">
          <Table className="min-w-[820px]">
            <TableHeader className="sticky top-0 z-10 bg-(--table-header) [&_th]:text-center">
              <TableRow>
                <TableHead rowSpan={2}>
                  {t('Supplier')} / {t('Model')}
                </TableHead>
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
              {channelGroups.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-muted-foreground h-24 text-center">
                    {t('No data')}
                  </TableCell>
                </TableRow>
              ) : (
                channelGroups.map((group) => {
                  const expanded = expandedIds.has(group.channelId)
                  return (
                    <ChannelRows
                      key={group.channelId}
                      group={group}
                      expanded={expanded}
                      onToggle={() => toggle(group.channelId)}
                    />
                  )
                })
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

function ChannelRows(props: {
  group: ReturnType<typeof groupReconciliationRowsByChannel>[number]
  expanded: boolean
  onToggle: () => void
}) {
  const group = props.group
  // 差异只反映可对账子行；全部不可对账时如实显示无数据
  const fullyUnreconciled = group.unreconciledRequests >= group.requestCount
  const inputDelta = group.upstreamInputTokens - group.reconciledSystemInputTokens
  const highlighted = group.estimated || group.local

  return (
    <>
      <TableRow
        className={highlighted ? 'bg-amber-500/5' : undefined}
        onClick={props.onToggle}
      >
        <TableCell>
          <button
            type="button"
            aria-expanded={props.expanded}
            onClick={(event) => {
              event.stopPropagation()
              props.onToggle()
            }}
            className="flex items-center gap-1 rounded-sm text-left font-medium hover:underline"
          >
            {props.expanded ? (
              <ChevronDown className="size-4 shrink-0" aria-hidden="true" />
            ) : (
              <ChevronRight className="size-4 shrink-0" aria-hidden="true" />
            )}
            {group.channelName || `#${group.channelId}`}
          </button>
        </TableCell>
        <TableCell className="font-mono">
          {formatTokenCount(group.requestCount)}
        </TableCell>
        {fullyUnreconciled ? (
          <>
            <UpstreamPendingCell />
            <TableCell className="font-mono">
              {formatTokenCount(group.systemInputTokens)}
            </TableCell>
            <UpstreamPendingCell />
          </>
        ) : (
          <>
            <TableCell className="font-mono">
              {formatTokenCount(group.upstreamInputTokens)}
            </TableCell>
            <TableCell className="font-mono">
              {formatTokenCount(group.reconciledSystemInputTokens)}
            </TableCell>
            <TableCell className={`font-mono ${deltaColorClass(inputDelta)}`}>
              {formatDelta(inputDelta)}
            </TableCell>
          </>
        )}
        <TableCell className="font-mono">
          {formatTokenCount(group.systemOutputTokens)}
        </TableCell>
      </TableRow>
      {props.expanded
        ? group.children.map((row) => {
            const key = `${row.usage_billing_path}-${row.semantic}-${row.model_name}`
            return <ModelRow key={key} row={row} />
          })
        : null}
    </>
  )
}

function UpstreamPendingCell() {
  return <TableCell className="text-muted-foreground font-mono">—</TableCell>
}

function ModelRow(props: { row: UsageReconciliationRow }) {
  const row = props.row
  const inputDelta = computeRowInputDelta(row)
  const highlighted = row.estimated || row.local

  return (
    <TableRow className={highlighted ? 'bg-amber-500/5' : undefined}>
      <TableCell className="pl-8">
        <span className="font-mono text-xs">{row.model_name || '—'}</span>
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
          <TableCell className={`font-mono ${deltaColorClass(inputDelta)}`}>
            {formatDelta(inputDelta)}
          </TableCell>
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
  )
}
