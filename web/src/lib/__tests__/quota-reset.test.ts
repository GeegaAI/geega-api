import { describe, expect, it } from 'vitest'

import {
  displayQuotaResetMode,
  getQuotaEffectiveTimeLock,
  QUOTA_RESET_MODE_CYCLE,
  QUOTA_RESET_MODE_LABEL_KEYS,
  QUOTA_RESET_MODE_MONTHLY,
  QUOTA_RESET_MODE_NEVER,
  QUOTA_RESET_MODE_OPTIONS,
} from '../quota-reset'

describe('quota reset mode options', () => {
  it('offers exactly the three selectable policies (no follow-default entry)', () => {
    expect([...QUOTA_RESET_MODE_OPTIONS]).toEqual([
      QUOTA_RESET_MODE_MONTHLY,
      QUOTA_RESET_MODE_CYCLE,
      QUOTA_RESET_MODE_NEVER,
    ])
    expect(Object.keys(QUOTA_RESET_MODE_LABEL_KEYS).sort()).toEqual(
      [QUOTA_RESET_MODE_CYCLE, QUOTA_RESET_MODE_MONTHLY, QUOTA_RESET_MODE_NEVER].sort()
    )
  })

  it('never returns an empty label key for a selectable mode', () => {
    for (const labelKey of Object.values(QUOTA_RESET_MODE_LABEL_KEYS)) {
      expect(labelKey.length).toBeGreaterThan(0)
    }
  })

  it('shows unset users as monthly fixed-day reset', () => {
    expect(displayQuotaResetMode('')).toBe(QUOTA_RESET_MODE_MONTHLY)
    expect(displayQuotaResetMode(undefined)).toBe(QUOTA_RESET_MODE_MONTHLY)
    expect(displayQuotaResetMode('unknown')).toBe(QUOTA_RESET_MODE_MONTHLY)
    expect(displayQuotaResetMode(QUOTA_RESET_MODE_CYCLE)).toBe(
      QUOTA_RESET_MODE_CYCLE
    )
    expect(displayQuotaResetMode(QUOTA_RESET_MODE_NEVER)).toBe(
      QUOTA_RESET_MODE_NEVER
    )
  })
})

describe('quota effective time lock', () => {
  const now = 1_700_000_000

  it('locks while the cycle is active, even if quota was used up', () => {
    expect(
      getQuotaEffectiveTimeLock({ quota: 100, nextResetAt: now + 60, now })
    ).toBe('active-cycle')
    expect(
      getQuotaEffectiveTimeLock({ quota: 0, nextResetAt: now + 60, now })
    ).toBe('active-cycle')
  })

  it('locks until the user has quota', () => {
    expect(getQuotaEffectiveTimeLock({ quota: 0, nextResetAt: 0, now })).toBe(
      'no-quota'
    )
    expect(
      getQuotaEffectiveTimeLock({ quota: -10, nextResetAt: now - 1, now })
    ).toBe('no-quota')
  })

  it('allows editing once the previous cycle ended and quota is present', () => {
    expect(getQuotaEffectiveTimeLock({ quota: 100, nextResetAt: 0, now })).toBe(
      null
    )
    expect(
      getQuotaEffectiveTimeLock({ quota: 100, nextResetAt: now, now })
    ).toBe(null)
  })
})
