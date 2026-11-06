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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  formatAvailabilityCompensationDates,
  isAvailabilityCompensationRangeInvalid,
  resolveAvailabilityCompensationRange,
} from '../lib/range'

describe('resolveAvailabilityCompensationRange', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-23T10:30:00+08:00'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('resolves the current natural month for thisMonth', () => {
    const range = resolveAvailabilityCompensationRange(
      'thisMonth',
      '2026-10-23',
      '2026-10-23'
    )
    expect(range.startTimestamp).toBe(
      dayjs('2026-10-01T00:00:00+08:00').unix()
    )
    expect(range.endTimestamp).toBe(
      dayjs('2026-11-01T00:00:00+08:00').unix()
    )
  })

  it('resolves the previous natural month for lastMonth', () => {
    const range = resolveAvailabilityCompensationRange(
      'lastMonth',
      '2026-10-23',
      '2026-10-23'
    )
    expect(range.startTimestamp).toBe(
      dayjs('2026-09-01T00:00:00+08:00').unix()
    )
    expect(range.endTimestamp).toBe(
      dayjs('2026-10-01T00:00:00+08:00').unix()
    )
  })

  it('treats the custom end date as inclusive', () => {
    const range = resolveAvailabilityCompensationRange(
      'custom',
      '2026-09-15',
      '2026-09-30'
    )
    expect(range.startTimestamp).toBe(
      dayjs('2026-09-15T00:00:00+08:00').unix()
    )
    expect(range.endTimestamp).toBe(
      dayjs('2026-10-01T00:00:00+08:00').unix()
    )
  })
})

describe('formatAvailabilityCompensationDates', () => {
  it('renders the exclusive end timestamp as the previous day', () => {
    expect(
      formatAvailabilityCompensationDates({
        startTimestamp: dayjs('2026-09-01T00:00:00+08:00').unix(),
        endTimestamp: dayjs('2026-10-01T00:00:00+08:00').unix(),
      })
    ).toEqual({ startDate: '2026-09-01', endDate: '2026-09-30' })
  })
})

describe('isAvailabilityCompensationRangeInvalid', () => {
  it('rejects empty, reversed and oversized ranges', () => {
    expect(isAvailabilityCompensationRangeInvalid('', '')).toBe(true)
    expect(
      isAvailabilityCompensationRangeInvalid('2026-09-30', '2026-09-01')
    ).toBe(true)
    expect(
      isAvailabilityCompensationRangeInvalid('2026-07-01', '2026-09-02')
    ).toBe(true)
  })

  it('accepts a range of at most 62 days', () => {
    expect(
      isAvailabilityCompensationRangeInvalid('2026-09-01', '2026-09-30')
    ).toBe(false)
    expect(
      isAvailabilityCompensationRangeInvalid('2026-08-01', '2026-10-01')
    ).toBe(false)
  })
})
