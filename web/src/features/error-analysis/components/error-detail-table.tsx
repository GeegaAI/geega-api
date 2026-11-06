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
import { Link } from '@tanstack/react-router'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Fragment, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
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
import {
  formatErrorCount,
  formatErrorDateTime,
  formatErrorRate,
  getChannelDisplayName,
  getErrorRateTextClass,
} from '../lib/format'
import { buildErrorProviderTree } from '../lib/provider-tree'
import type { ErrorAnalysisRange } from '../lib/range'
import type { ErrorBreakdownRow, ErrorChannelStat } from '../types'
import { ErrorSampleMessage, ErrorStatusCodes } from './error-table-cells'

interface ErrorDetailTableProps {
  rows: ErrorChannelStat[]
  breakdownRows: ErrorBreakdownRow[]
  rowsTruncated: boolean
  range: ErrorAnalysisRange
}

const TABLE_WRAPPER_CLASS =
  'max-h-[640px] overflow-auto [&_[data-slot=table-container]]:overflow-visible [&_table]:border-separate [&_table]:border-spacing-0 [&_td]:border-r [&_td]:border-b [&_td:first-child]:border-l [&_th]:border-r [&_th]:border-b [&_thead_tr:first-child_th]:border-t [&_thead_tr:first-child_th:first-child]:border-l'

export function ErrorDetailTable(props: ErrorDetailTableProps) {
  const { t } = useTranslation()
  const [expandedKeys, setExpandedKeys] = useState<ReadonlySet<string>>(
    () => new Set()
  )

  const groups = useMemo(
    () => buildErrorProviderTree(props.rows, props.breakdownRows),
    [props.rows, props.breakdownRows]
  )

  const allExpandableKeys = useMemo(() => {
    const keys: string[] = []
    for (const group of groups) {
      keys.push(group.key)
      for (const entry of group.channels) {
        keys.push(`channel-${entry.channel.channel_id}`)
      }
    }
    return keys
  }, [groups])

  const allExpanded =
    allExpandableKeys.length > 0 &&
    allExpandableKeys.every((key) => expandedKeys.has(key))

  const toggleKey = (key: string) => {
    setExpandedKeys((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const handleToggleAll = () => {
    setExpandedKeys(allExpanded ? new Set() : new Set(allExpandableKeys))
  }

  const renderExpandButton = (key: string, expanded: boolean) => (
    <Button
      variant="ghost"
      size="sm"
      className="h-6 w-6 shrink-0 p-0"
      aria-expanded={expanded}
      aria-label={expanded ? t('Collapse') : t('Expand')}
      onClick={(event) => {
        event.stopPropagation()
        toggleKey(key)
      }}
    >
      {expanded ? (
        <ChevronDown className="size-4" />
      ) : (
        <ChevronRight className="size-4" />
      )}
    </Button>
  )

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">{t('Error Breakdown')}</CardTitle>
          <Button
            variant="outline"
            size="sm"
            disabled={allExpandableKeys.length === 0}
            onClick={handleToggleAll}
          >
            {allExpanded ? t('Collapse All') : t('Expand All')}
          </Button>
        </div>
        <p className="text-muted-foreground text-xs">
          {t('Grouped by provider, channel and model.')}
        </p>
      </CardHeader>
      <CardContent>
        {props.rowsTruncated ? (
          <p className="text-muted-foreground mb-2 text-xs">
            {t(
              'Only the top 500 channel / model / reason combinations are shown.'
            )}
          </p>
        ) : null}
        <div className={TABLE_WRAPPER_CLASS}>
          <Table className="min-w-[1320px]">
            <TableHeader className="sticky top-0 z-10 bg-(--table-header) [&_th]:text-center">
              <TableRow>
                <TableHead className="text-left">
                  {t('Provider / Channel / Model')}
                </TableHead>
                <TableHead>{t('Error Count')}</TableHead>
                <TableHead>{t('Request Count')}</TableHead>
                <TableHead>{t('Error Rate')}</TableHead>
                <TableHead className="text-left">{t('Top Reason')}</TableHead>
                <TableHead>{t('Status Code')}</TableHead>
                <TableHead>{t('Models')}</TableHead>
                <TableHead>{t('Affected Users')}</TableHead>
                <TableHead>{t('Last Error')}</TableHead>
                <TableHead className="text-left">
                  {t('Sample Message')}
                </TableHead>
                <TableHead>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={11}
                    className="text-muted-foreground h-24 text-center"
                  >
                    {t('No data')}
                  </TableCell>
                </TableRow>
              ) : (
                groups.map((group) => {
                  const groupExpanded = expandedKeys.has(group.key)
                  return (
                    <Fragment key={group.key}>
                      <TableRow
                        className="cursor-pointer"
                        onClick={() => toggleKey(group.key)}
                      >
                        <TableCell>
                          <div className="flex items-center gap-1.5">
                            {renderExpandButton(group.key, groupExpanded)}
                            <span className="font-medium">
                              {group.known_type
                                ? group.type_label
                                : t('Unknown')}
                            </span>
                            <span className="text-muted-foreground font-mono text-[11px]">
                              ({t('Channels')}: {group.channels.length})
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="text-center font-mono">
                          {formatErrorCount(group.error_count)}
                        </TableCell>
                        <TableCell className="text-center font-mono">
                          {formatErrorCount(group.request_count)}
                        </TableCell>
                        <TableCell
                          className={`text-center font-mono ${getErrorRateTextClass(group.error_rate)}`}
                        >
                          {formatErrorRate(group.error_rate)}
                        </TableCell>
                        <TableCell>
                          {group.top_reason_code
                            ? getErrorReasonLabel(group.top_reason_code, t)
                            : '—'}
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-center">
                            <ErrorStatusCodes codes={group.status_codes} />
                          </div>
                        </TableCell>
                        <TableCell className="text-center font-mono">
                          {formatErrorCount(group.model_count)}
                        </TableCell>
                        <TableCell className="text-muted-foreground text-center">
                          —
                        </TableCell>
                        <TableCell className="text-center font-mono">
                          {formatErrorDateTime(group.last_error_at)}
                        </TableCell>
                        <TableCell>
                          <span className="text-muted-foreground">—</span>
                        </TableCell>
                        <TableCell>
                          <span className="text-muted-foreground">—</span>
                        </TableCell>
                      </TableRow>
                      {groupExpanded
                        ? group.channels.map((entry) => {
                            const channel = entry.channel
                            const channelKey = `channel-${channel.channel_id}`
                            const channelExpanded =
                              expandedKeys.has(channelKey)
                            return (
                              <Fragment key={channelKey}>
                                <TableRow
                                  className="cursor-pointer"
                                  onClick={() => toggleKey(channelKey)}
                                >
                                  <TableCell className="pl-9">
                                    <div className="flex items-center gap-1.5">
                                      {renderExpandButton(
                                        channelKey,
                                        channelExpanded
                                      )}
                                      <div>
                                        <div className="font-medium">
                                          {getChannelDisplayName(
                                            channel.channel_id,
                                            channel.channel_name,
                                            t('Unknown Channel')
                                          )}
                                        </div>
                                        {channel.channel_name ? (
                                          <div className="text-muted-foreground font-mono text-[11px]">
                                            #{channel.channel_id}
                                          </div>
                                        ) : null}
                                      </div>
                                    </div>
                                  </TableCell>
                                  <TableCell className="text-center font-mono">
                                    {formatErrorCount(channel.error_count)}
                                  </TableCell>
                                  <TableCell className="text-center font-mono">
                                    {formatErrorCount(channel.request_count)}
                                  </TableCell>
                                  <TableCell
                                    className={`text-center font-mono ${getErrorRateTextClass(channel.error_rate)}`}
                                  >
                                    {formatErrorRate(channel.error_rate)}
                                  </TableCell>
                                  <TableCell>
                                    {channel.top_reason_code
                                      ? getErrorReasonLabel(
                                          channel.top_reason_code,
                                          t
                                        )
                                      : '—'}
                                  </TableCell>
                                  <TableCell>
                                    <span className="text-muted-foreground">
                                      —
                                    </span>
                                  </TableCell>
                                  <TableCell className="text-center font-mono">
                                    {formatErrorCount(channel.model_count)}
                                  </TableCell>
                                  <TableCell className="text-center font-mono">
                                    {formatErrorCount(channel.user_count)}
                                  </TableCell>
                                  <TableCell className="font-mono text-xs">
                                    {formatErrorDateTime(channel.last_error_at)}
                                  </TableCell>
                                  <TableCell>
                                    <span className="text-muted-foreground">
                                      —
                                    </span>
                                  </TableCell>
                                  <TableCell>
                                    {channel.channel_id !== 0 ? (
                                      <Link
                                        to="/usage-logs/$section"
                                        params={{ section: 'common' }}
                                        search={{
                                          type: ['5'],
                                          channel: String(channel.channel_id),
                                          startTime:
                                            props.range.startTimestamp * 1000,
                                          endTime:
                                            props.range.endTimestamp * 1000,
                                          page: 1,
                                        }}
                                        onClick={(event) =>
                                          event.stopPropagation()
                                        }
                                        className="text-primary text-xs hover:underline"
                                      >
                                        {t('View Logs')}
                                      </Link>
                                    ) : (
                                      <span className="text-muted-foreground">
                                        —
                                      </span>
                                    )}
                                  </TableCell>
                                </TableRow>
                                {channelExpanded
                                  ? entry.models.map((model) => (
                                      <TableRow
                                        key={`${channel.channel_id}-${model.model_name}`}
                                      >
                                        <TableCell className="pl-16">
                                          <span className="font-mono text-xs">
                                            {model.model_name ||
                                              t('Unknown Model')}
                                          </span>
                                        </TableCell>
                                        <TableCell className="text-center font-mono">
                                          {formatErrorCount(model.error_count)}
                                        </TableCell>
                                        <TableCell className="text-center font-mono">
                                          {formatErrorCount(
                                            model.request_count
                                          )}
                                        </TableCell>
                                        <TableCell
                                          className={`text-center font-mono ${getErrorRateTextClass(model.error_rate)}`}
                                          title={`${model.error_count} / ${model.request_count}`}
                                        >
                                          {formatErrorRate(model.error_rate)}
                                        </TableCell>
                                        <TableCell>
                                          {model.top_reason_code
                                            ? getErrorReasonLabel(
                                                model.top_reason_code,
                                                t
                                              )
                                            : '—'}
                                        </TableCell>
                                        <TableCell>
                                          <div className="flex justify-center">
                                            <ErrorStatusCodes
                                              codes={model.status_codes}
                                            />
                                          </div>
                                        </TableCell>
                                        <TableCell className="text-muted-foreground text-center">
                                          —
                                        </TableCell>
                                        <TableCell className="text-muted-foreground text-center">
                                          —
                                        </TableCell>
                                        <TableCell className="text-muted-foreground text-center">
                                          —
                                        </TableCell>
                                        <TableCell>
                                          <ErrorSampleMessage
                                            message={model.sample_message}
                                          />
                                        </TableCell>
                                        <TableCell>
                                          {model.model_name ? (
                                            <Link
                                              to="/usage-logs/$section"
                                              params={{ section: 'common' }}
                                              search={{
                                                type: ['5'],
                                                channel: String(
                                                  channel.channel_id
                                                ),
                                                model: model.model_name,
                                                startTime:
                                                  props.range.startTimestamp *
                                                  1000,
                                                endTime:
                                                  props.range.endTimestamp *
                                                  1000,
                                                page: 1,
                                              }}
                                              className="text-primary text-xs hover:underline"
                                            >
                                              {t('View Logs')}
                                            </Link>
                                          ) : (
                                            <span className="text-muted-foreground">
                                              —
                                            </span>
                                          )}
                                        </TableCell>
                                      </TableRow>
                                    ))
                                  : null}
                              </Fragment>
                            )
                          })
                        : null}
                    </Fragment>
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
