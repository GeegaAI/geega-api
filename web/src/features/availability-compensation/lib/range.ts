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

export type AvailabilityCompensationRangePreset =
  | 'thisMonth'
  | 'lastMonth'
  | 'custom'

export interface ResolvedAvailabilityCompensationRange {
  startTimestamp: number
  // 开区间：结束日的次日 00:00
  endTimestamp: number
}

export const AVAILABILITY_COMPENSATION_MAX_RANGE_DAYS = 62

function resolveMonthRange(month: dayjs.Dayjs): ResolvedAvailabilityCompensationRange {
  return {
    startTimestamp: month.startOf('month').unix(),
    endTimestamp: month.add(1, 'month').startOf('month').unix(),
  }
}

export function resolveAvailabilityCompensationRange(
  preset: AvailabilityCompensationRangePreset,
  startDate: string,
  endDate: string
): ResolvedAvailabilityCompensationRange {
  const now = dayjs()
  if (preset === 'thisMonth') {
    return resolveMonthRange(now)
  }
  if (preset === 'lastMonth') {
    return resolveMonthRange(now.subtract(1, 'month'))
  }
  const start = dayjs(startDate).startOf('day')
  const end = dayjs(endDate).add(1, 'day').startOf('day')
  if (!start.isValid() || !end.isValid() || !start.isBefore(end)) {
    return resolveMonthRange(now)
  }
  return {
    startTimestamp: start.unix(),
    endTimestamp: end.unix(),
  }
}

export function isAvailabilityCompensationRangeInvalid(
  startDate: string,
  endDate: string
): boolean {
  if (!startDate || !endDate || startDate > endDate) {
    return true
  }
  const start = dayjs(startDate).startOf('day')
  const end = dayjs(endDate).add(1, 'day').startOf('day')
  if (!start.isValid() || !end.isValid()) {
    return true
  }
  return end.diff(start, 'day', true) > AVAILABILITY_COMPENSATION_MAX_RANGE_DAYS
}

export function formatAvailabilityCompensationDates(
  range: ResolvedAvailabilityCompensationRange
): { startDate: string; endDate: string } {
  return {
    startDate: dayjs.unix(range.startTimestamp).format('YYYY-MM-DD'),
    endDate: dayjs.unix(range.endTimestamp).subtract(1, 'second').format('YYYY-MM-DD'),
  }
}
