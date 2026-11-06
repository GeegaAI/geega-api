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
  compensationFailCount,
  countCompensatedSuppliers,
  groupCompensationRowsBySupplier,
  sortCompensationRows,
} from '../lib/derive'
import type {
  AvailabilityCompensationRow,
  AvailabilityCompensationSupplierRow,
} from '../types'

function makeRow(
  overrides: Partial<AvailabilityCompensationRow>
): AvailabilityCompensationRow {
  return {
    model_name: 'model-a',
    channel_id: 1,
    channel_name: 'chan-1',
    supplier: 'acme',
    ok_requests: 0,
    fail_all_requests: 0,
    fail_service_requests: 0,
    error_rate_all: 0,
    error_rate_service: 0,
    error_minutes_ge50_all: 0,
    error_minutes_any_all: 0,
    error_minutes_ge50_service: 0,
    error_minutes_any_service: 0,
    availability_ge50_all: 100,
    availability_any_all: 100,
    availability_ge50_service: 100,
    availability_any_service: 100,
    tier_all: 'none',
    tier_service: 'none',
    compensation_pct_all: 0,
    compensation_pct_service: 0,
    ...overrides,
  }
}

function makeSupplier(
  overrides: Partial<AvailabilityCompensationSupplierRow>
): AvailabilityCompensationSupplierRow {
  return {
    supplier: 'acme',
    channel_count: 1,
    ok_requests: 0,
    fail_all_requests: 0,
    fail_service_requests: 0,
    error_rate_all: 0,
    error_rate_service: 0,
    error_minutes_ge50_all: 0,
    error_minutes_any_all: 0,
    error_minutes_ge50_service: 0,
    error_minutes_any_service: 0,
    availability_ge50_all: 100,
    availability_any_all: 100,
    availability_ge50_service: 100,
    availability_any_service: 100,
    tier_all: 'none',
    tier_service: 'none',
    compensation_pct_all: 0,
    compensation_pct_service: 0,
    ...overrides,
  }
}

describe('sortCompensationRows', () => {
  it('orders rows by fail count desc, then channel id asc, then model name', () => {
    const rows = [
      makeRow({ channel_id: 2, model_name: 'model-b', fail_all_requests: 5 }),
      makeRow({ channel_id: 1, model_name: 'model-z', fail_all_requests: 5 }),
      makeRow({ channel_id: 1, model_name: 'model-a', fail_all_requests: 9 }),
    ]

    const sorted = sortCompensationRows(rows, 'all')

    expect(sorted.map((row) => row.model_name)).toEqual([
      'model-a',
      'model-z',
      'model-b',
    ])
  })

  it('uses service-side fail counts when version is service', () => {
    const rows = [
      makeRow({
        channel_id: 1,
        model_name: 'model-client',
        fail_all_requests: 9,
        fail_service_requests: 0,
      }),
      makeRow({
        channel_id: 2,
        model_name: 'model-server',
        fail_all_requests: 1,
        fail_service_requests: 4,
      }),
    ]

    const sorted = sortCompensationRows(rows, 'service')

    expect(sorted.map((row) => row.model_name)).toEqual([
      'model-server',
      'model-client',
    ])
  })

  it('does not mutate the input rows array', () => {
    const rows = [
      makeRow({ channel_id: 1, fail_all_requests: 1 }),
      makeRow({ channel_id: 2, fail_all_requests: 2 }),
    ]

    sortCompensationRows(rows, 'all')

    expect(rows.map((row) => row.channel_id)).toEqual([1, 2])
  })
})

describe('compensationFailCount', () => {
  it('returns all-error counts for version all and service-side counts for version service', () => {
    const supplier = makeSupplier({
      fail_all_requests: 7,
      fail_service_requests: 3,
    })

    expect(compensationFailCount(supplier, 'all')).toBe(7)
    expect(compensationFailCount(supplier, 'service')).toBe(3)
  })
})

describe('countCompensatedSuppliers', () => {
  it('counts suppliers with a positive compensation ratio for the active version', () => {
    const suppliers = [
      makeSupplier({
        supplier: 'v1-only',
        compensation_pct_all: 5,
        compensation_pct_service: 0,
      }),
      makeSupplier({
        supplier: 'both',
        compensation_pct_all: 8,
        compensation_pct_service: 10,
      }),
      makeSupplier({
        supplier: 'none',
        compensation_pct_all: 0,
        compensation_pct_service: 0,
      }),
    ]

    expect(countCompensatedSuppliers(suppliers, 'all')).toBe(2)
    expect(countCompensatedSuppliers(suppliers, 'service')).toBe(1)
  })
})

describe('groupCompensationRowsBySupplier', () => {
  it('groups detail rows under their supplier and orders suppliers by fail count desc', () => {
    const suppliers = [
      makeSupplier({ supplier: 'acme', fail_all_requests: 2 }),
      makeSupplier({ supplier: 'beta', fail_all_requests: 8 }),
    ]
    const rows = [
      makeRow({ supplier: 'acme', channel_id: 1 }),
      makeRow({ supplier: 'beta', channel_id: 2 }),
      makeRow({ supplier: 'acme', channel_id: 3 }),
    ]

    const groups = groupCompensationRowsBySupplier(rows, suppliers, 'all')

    expect(groups.map((group) => group.supplier.supplier)).toEqual([
      'beta',
      'acme',
    ])
    expect(groups[0].rows).toHaveLength(1)
    expect(groups[1].rows).toHaveLength(2)
  })

  it('reorders suppliers when the active version changes their fail counts', () => {
    const suppliers = [
      makeSupplier({
        supplier: 'acme',
        fail_all_requests: 8,
        fail_service_requests: 1,
      }),
      makeSupplier({
        supplier: 'beta',
        fail_all_requests: 2,
        fail_service_requests: 6,
      }),
    ]
    const rows = [
      makeRow({ supplier: 'acme', channel_id: 1 }),
      makeRow({ supplier: 'beta', channel_id: 2 }),
    ]

    const allGroups = groupCompensationRowsBySupplier(rows, suppliers, 'all')
    const serviceGroups = groupCompensationRowsBySupplier(
      rows,
      suppliers,
      'service'
    )

    expect(allGroups.map((group) => group.supplier.supplier)).toEqual([
      'acme',
      'beta',
    ])
    expect(serviceGroups.map((group) => group.supplier.supplier)).toEqual([
      'beta',
      'acme',
    ])
  })
})
