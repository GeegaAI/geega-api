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
import dayjs from 'dayjs'

export type ErrorAnalysisRangePreset =
  | 'today'
  | 'yesterday'
  | 'last7days'
  | 'custom'

export interface ErrorAnalysisRange {
  startTimestamp: number
  endTimestamp: number
}

export function resolveErrorAnalysisRange(
  preset: ErrorAnalysisRangePreset,
  startDate: string,
  endDate: string
): ErrorAnalysisRange {
  const now = dayjs()
  if (preset === 'today') {
    return {
      startTimestamp: now.startOf('day').unix(),
      endTimestamp: now.unix(),
    }
  }
  if (preset === 'yesterday') {
    const yesterday = now.subtract(1, 'day')
    return {
      startTimestamp: yesterday.startOf('day').unix(),
      endTimestamp: yesterday.endOf('day').unix(),
    }
  }
  if (preset === 'last7days') {
    return {
      startTimestamp: now.subtract(6, 'day').startOf('day').unix(),
      endTimestamp: now.unix(),
    }
  }
  return {
    startTimestamp: dayjs(startDate).startOf('day').unix(),
    endTimestamp: dayjs(endDate).endOf('day').unix(),
  }
}
