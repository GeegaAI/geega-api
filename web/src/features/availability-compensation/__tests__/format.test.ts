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
import { describe, expect, it } from 'vitest'

import {
  formatAvailabilityCompensationCount,
  formatAvailabilityCompensationPercent,
  formatAvailabilityCompensationRatio,
  hasCompensation,
  tierLabelKey,
} from '../lib/format'

describe('availability compensation formatting', () => {
  it('formats counts and percentages', () => {
    expect(formatAvailabilityCompensationCount(1234567)).toBe('1,234,567')
    expect(formatAvailabilityCompensationPercent(99.91234)).toBe('99.912%')
    expect(formatAvailabilityCompensationPercent(100)).toBe('100.000%')
  })

  it('hides a zero compensation ratio', () => {
    expect(formatAvailabilityCompensationRatio(0)).toBe('—')
    expect(formatAvailabilityCompensationRatio(5)).toBe('5%')
    expect(formatAvailabilityCompensationRatio(10)).toBe('10%')
  })

  it('maps tiers to labels and compensation flags', () => {
    expect(tierLabelKey('none')).toBe('No Compensation')
    expect(tierLabelKey('tier_99')).toBe('99%≤Availability≤99.9%')
    expect(tierLabelKey('tier_95')).toBe('95%≤Availability<99%')
    expect(tierLabelKey('below_95')).toBe('Availability<95%')
    expect(hasCompensation('none')).toBe(false)
    expect(hasCompensation('tier_99')).toBe(true)
  })
})
