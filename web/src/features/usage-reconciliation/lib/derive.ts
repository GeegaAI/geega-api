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
import type { UsageReconciliationRow } from '../types'

// 对账核心是输入侧：上游计费口径输入 vs 系统记录输入。
// 系统输出即上游 usage 的照抄值，没有独立上游输出口径，不做输出对比。
export function computeRowInputDelta(row: UsageReconciliationRow): number {
  return row.upstream_input_tokens - row.system_input_tokens
}

// 后端已标注可对账性：语义已知且非估算/本地；不可对账行的 upstream 字段为 0，
// 上游与差异数值只能来自可对账行，避免"上游=系统"的默认假象
export function isReconcilable(row: UsageReconciliationRow): boolean {
  return row.reconcilable
}

export function computeReconcilableShare(rows: UsageReconciliationRow[]): number {
  const totalRequests = rows.reduce((sum, row) => sum + row.request_count, 0)
  if (totalRequests === 0) {
    return 0
  }
  const reconciledRequests = rows
    .filter((row) => isReconcilable(row))
    .reduce((sum, row) => sum + row.request_count, 0)
  return reconciledRequests / totalRequests
}

export function formatTokenCount(value: number): string {
  return value.toLocaleString('en-US')
}

export function formatDelta(value: number): string {
  const formatted = formatTokenCount(Math.abs(value))
  if (value > 0) {
    return `+${formatted}`
  }
  if (value < 0) {
    return `-${formatted}`
  }
  return formatted
}

export function formatPercent(fraction: number): string {
  return `${(fraction * 100).toFixed(1)}%`
}

export function deltaColorClass(value: number): string {
  if (value > 0) {
    return 'text-amber-600 dark:text-amber-400'
  }
  if (value < 0) {
    return 'text-emerald-600 dark:text-emerald-400'
  }
  return ''
}

export interface ReconciliationChannelGroup {
  channelId: number
  channelName: string
  estimated: boolean
  local: boolean
  requestCount: number
  unreconciledRequests: number
  quota: number
  systemInputTokens: number
  systemOutputTokens: number
  // 以下上游与差异只累计可对账子行
  reconciledSystemInputTokens: number
  upstreamInputTokens: number
  children: UsageReconciliationRow[]
}

// 后端按 来源×语义×渠道×模型 最细粒度返回；此处按 (来源, 语义) 聚合出口径视图
export function aggregateReconciliationRowsByPath(
  rows: UsageReconciliationRow[]
): UsageReconciliationRow[] {
  const merged = new Map<string, UsageReconciliationRow>()
  for (const row of rows) {
    const key = `${row.usage_billing_path}\n${row.semantic}`
    const target = merged.get(key)
    if (!target) {
      merged.set(key, { ...row })
      continue
    }
    target.reconcilable = target.reconcilable && row.reconcilable
    target.request_count += row.request_count
    target.quota += row.quota
    target.system_input_tokens += row.system_input_tokens
    target.system_output_tokens += row.system_output_tokens
    target.upstream_input_tokens += row.upstream_input_tokens
  }
  return [...merged.values()].sort((a, b) => b.quota - a.quota)
}

export function groupReconciliationRowsByChannel(
  rows: UsageReconciliationRow[]
): ReconciliationChannelGroup[] {
  const groups = new Map<number, ReconciliationChannelGroup>()
  for (const row of rows) {
    const key = row.channel_id
    let group = groups.get(key)
    if (!group) {
      group = {
        channelId: row.channel_id,
        channelName: row.channel_name,
        estimated: false,
        local: false,
        requestCount: 0,
        unreconciledRequests: 0,
        quota: 0,
        systemInputTokens: 0,
        systemOutputTokens: 0,
        reconciledSystemInputTokens: 0,
        upstreamInputTokens: 0,
        children: [],
      }
      groups.set(key, group)
    }
    group.estimated = group.estimated || row.estimated
    group.local = group.local || row.local
    group.requestCount += row.request_count
    group.quota += row.quota
    group.systemInputTokens += row.system_input_tokens
    group.systemOutputTokens += row.system_output_tokens
    if (isReconcilable(row)) {
      group.reconciledSystemInputTokens += row.system_input_tokens
      group.upstreamInputTokens += row.upstream_input_tokens
    } else {
      group.unreconciledRequests += row.request_count
    }
    group.children.push(row)
  }
  const list = [...groups.values()]
  for (const group of list) {
    group.children.sort((a, b) => b.quota - a.quota)
  }
  return list.sort((a, b) => b.quota - a.quota)
}
