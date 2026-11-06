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

import { AVAILABILITY_COMPENSATION_TIER_LABEL_KEYS } from '../constants'
import type { AvailabilityCompensationTier } from '../types'

const countFormatter = new Intl.NumberFormat('en-US')

export function formatAvailabilityCompensationCount(value: number): string {
  return countFormatter.format(value)
}

export function formatAvailabilityCompensationPercent(
  value: number,
  digits = 3
): string {
  return `${value.toFixed(digits)}%`
}

export function formatAvailabilityCompensationRatio(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return '—'
  }
  return `${value.toFixed(0)}%`
}

export function tierLabelKey(tier: AvailabilityCompensationTier): string {
  return AVAILABILITY_COMPENSATION_TIER_LABEL_KEYS[tier] ?? 'No Compensation'
}

export function hasCompensation(tier: AvailabilityCompensationTier): boolean {
  return tier !== 'none'
}

export function formatAvailabilityCompensationDateTime(
  timestamp: number
): string {
  if (!timestamp) {
    return '—'
  }
  return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm')
}
