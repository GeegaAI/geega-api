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

export type ReconciliationRangePreset =
  | 'today'
  | 'last7days'
  | 'last30days'
  | 'custom'

export interface ResolvedRange {
  startTimestamp: number
  endTimestamp: number
}

const FALLBACK_RANGE: () => ResolvedRange = () => ({
  startTimestamp: dayjs().subtract(6, 'day').startOf('day').unix(),
  endTimestamp: dayjs().unix(),
})

export function resolveReconciliationRange(
  preset: ReconciliationRangePreset,
  startDate: string,
  endDate: string
): ResolvedRange {
  const now = dayjs()
  switch (preset) {
    case 'today':
      return {
        startTimestamp: now.startOf('day').unix(),
        endTimestamp: now.unix(),
      }
    case 'last7days':
      return {
        startTimestamp: now.subtract(6, 'day').startOf('day').unix(),
        endTimestamp: now.unix(),
      }
    case 'last30days':
      return {
        startTimestamp: now.subtract(29, 'day').startOf('day').unix(),
        endTimestamp: now.unix(),
      }
    case 'custom': {
      const start = dayjs(startDate).startOf('day')
      const end = dayjs(endDate).endOf('day')
      if (!start.isValid() || !end.isValid() || start.isAfter(end)) {
        return FALLBACK_RANGE()
      }
      return {
        startTimestamp: start.unix(),
        endTimestamp: Math.min(end.unix(), now.unix()),
      }
    }
  }
}
