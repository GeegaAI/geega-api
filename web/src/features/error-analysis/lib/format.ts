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

const countFormatter = new Intl.NumberFormat()

export function formatErrorCount(value: number): string {
  return countFormatter.format(value)
}

export function formatErrorRate(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0%'
  const percent = value * 100
  if (percent < 0.01) return '<0.01%'
  return `${percent.toFixed(2)}%`
}

export function getErrorRateTextClass(value: number): string {
  if (value >= 0.05) return 'text-red-600 dark:text-red-400'
  if (value >= 0.01) return 'text-amber-600 dark:text-amber-400'
  return ''
}

export function formatTrendBucket(
  timestamp: number,
  bucket: 'hour' | 'day'
): string {
  return dayjs(timestamp * 1000).format(
    bucket === 'hour' ? 'MM-DD HH:00' : 'MM-DD'
  )
}

export function formatTrendTooltip(
  timestamp: number,
  bucket: 'hour' | 'day'
): string {
  return dayjs(timestamp * 1000).format(
    bucket === 'hour' ? 'MM-DD HH:00' : 'YYYY-MM-DD'
  )
}

export function formatErrorDateTime(timestamp: number): string {
  if (!timestamp) return '—'
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm')
}

export function getChannelDisplayName(
  channelId: number,
  channelName: string,
  unknownLabel: string
): string {
  if (channelName) return channelName
  if (channelId === 0) return unknownLabel
  return `#${channelId}`
}
