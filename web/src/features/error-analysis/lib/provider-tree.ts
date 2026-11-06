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
import { CHANNEL_TYPES } from '@/features/channels/constants'

import type {
  ErrorBreakdownRow,
  ErrorChannelGroupRow,
  ErrorChannelStat,
  ErrorModelDetailRow,
  ErrorProviderGroup,
} from '../types'

const UNKNOWN_PROVIDER_KEY = 'unknown'

function getProviderKey(channelType: number): string {
  return channelType in CHANNEL_TYPES ? `type-${channelType}` : UNKNOWN_PROVIDER_KEY
}

function getProviderLabel(channelType: number): string {
  return CHANNEL_TYPES[channelType as keyof typeof CHANNEL_TYPES] ?? ''
}

function pickTopReason(counts: Map<string, number>): string {
  let top = ''
  let topCount = -1
  for (const [code, count] of counts) {
    if (count > topCount || (count === topCount && code < top)) {
      top = code
      topCount = count
    }
  }
  return top
}

function unionStatusCodes(rows: ErrorBreakdownRow[]): number[] {
  const codes = new Set<number>()
  for (const row of rows) {
    for (const code of row.status_codes) {
      codes.add(code)
    }
  }
  return [...codes].sort((a, b) => a - b)
}

/**
 * Aggregate channel × model × reason breakdown rows into one row per model.
 * User counts are only exact per reason, so they are intentionally not
 * summed here.
 */
function buildModelRows(
  channelRows: ErrorBreakdownRow[]
): ErrorModelDetailRow[] {
  const byModel = new Map<string, ErrorBreakdownRow[]>()
  for (const row of channelRows) {
    const list = byModel.get(row.model_name)
    if (list) {
      list.push(row)
    } else {
      byModel.set(row.model_name, [row])
    }
  }

  const modelRows: ErrorModelDetailRow[] = []
  for (const [modelName, reasonRows] of byModel) {
    const reasonCounts = new Map<string, number>()
    let errorCount = 0
    for (const row of reasonRows) {
      errorCount += row.error_count
      reasonCounts.set(
        row.reason_code,
        (reasonCounts.get(row.reason_code) ?? 0) + row.error_count
      )
    }
    const topReasonCode = pickTopReason(reasonCounts)
    const topRow =
      reasonRows.find((row) => row.reason_code === topReasonCode) ??
      reasonRows[0]
    modelRows.push({
      model_name: modelName,
      error_count: errorCount,
      request_count: topRow.channel_model_request_count,
      error_rate: topRow.channel_model_error_rate,
      top_reason_code: topReasonCode,
      status_codes: unionStatusCodes(reasonRows),
      sample_message: topRow.sample_message,
    })
  }

  return modelRows.sort((a, b) => {
    if (a.error_count !== b.error_count) {
      return b.error_count - a.error_count
    }
    return a.model_name.localeCompare(b.model_name)
  })
}

interface ProviderGroupDraft {
  key: string
  channel_type: number
  type_label: string
  known_type: boolean
  channels: ErrorChannelGroupRow[]
  error_count: number
  request_count: number
  modelNames: Set<string>
  reasonCounts: Map<string, number>
  statusCodes: Set<number>
  last_error_at: number
}

/**
 * Group channel health stats and per-model breakdown rows into a
 * provider (channel type) → channel → model tree for the drill-down
 * detail table.
 */
export function buildErrorProviderTree(
  channels: ErrorChannelStat[],
  breakdownRows: ErrorBreakdownRow[]
): ErrorProviderGroup[] {
  const rowsByChannel = new Map<number, ErrorBreakdownRow[]>()
  for (const row of breakdownRows) {
    const list = rowsByChannel.get(row.channel_id)
    if (list) {
      list.push(row)
    } else {
      rowsByChannel.set(row.channel_id, [row])
    }
  }

  const groups = new Map<string, ProviderGroupDraft>()
  for (const channel of channels) {
    const key = getProviderKey(channel.channel_type)
    let group = groups.get(key)
    if (!group) {
      group = {
        key,
        channel_type: channel.channel_type,
        type_label: getProviderLabel(channel.channel_type),
        known_type: channel.channel_type in CHANNEL_TYPES,
        channels: [],
        error_count: 0,
        request_count: 0,
        modelNames: new Set<string>(),
        reasonCounts: new Map<string, number>(),
        statusCodes: new Set<number>(),
        last_error_at: 0,
      }
      groups.set(key, group)
    }

    const channelRows = rowsByChannel.get(channel.channel_id) ?? []
    group.channels.push({
      channel,
      models: buildModelRows(channelRows),
    })
    group.error_count += channel.error_count
    group.request_count += channel.request_count
    for (const row of channelRows) {
      group.modelNames.add(row.model_name)
      group.reasonCounts.set(
        row.reason_code,
        (group.reasonCounts.get(row.reason_code) ?? 0) + row.error_count
      )
      for (const code of row.status_codes) {
        group.statusCodes.add(code)
      }
    }
    if (channel.last_error_at > group.last_error_at) {
      group.last_error_at = channel.last_error_at
    }
  }

  const result: ErrorProviderGroup[] = []
  for (const group of groups.values()) {
    result.push({
      key: group.key,
      channel_type: group.channel_type,
      type_label: group.type_label,
      known_type: group.known_type,
      channels: group.channels.sort((a, b) =>
        a.channel.error_count !== b.channel.error_count
          ? b.channel.error_count - a.channel.error_count
          : a.channel.channel_id - b.channel.channel_id
      ),
      error_count: group.error_count,
      request_count: group.request_count,
      error_rate:
        group.request_count > 0
          ? group.error_count / group.request_count
          : 0,
      model_count: group.modelNames.size,
      top_reason_code: pickTopReason(group.reasonCounts),
      status_codes: [...group.statusCodes].sort((a, b) => a - b),
      last_error_at: group.last_error_at,
    })
  }

  return result.sort((a, b) => {
    if (a.error_count !== b.error_count) {
      return b.error_count - a.error_count
    }
    return a.key.localeCompare(b.key)
  })
}
