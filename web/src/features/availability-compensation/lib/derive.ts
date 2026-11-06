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
import type {
  AvailabilityCompensationRow,
  AvailabilityCompensationSupplierRow,
  AvailabilityCompensationVersion,
} from '../types'

export interface CompensationRowGroup {
  supplier: AvailabilityCompensationSupplierRow
  rows: AvailabilityCompensationRow[]
}

export function compensationFailCount(
  row: AvailabilityCompensationRow | AvailabilityCompensationSupplierRow,
  version: AvailabilityCompensationVersion
): number {
  return version === 'all' ? row.fail_all_requests : row.fail_service_requests
}

export function sortCompensationRows(
  rows: AvailabilityCompensationRow[],
  version: AvailabilityCompensationVersion
): AvailabilityCompensationRow[] {
  const sorted = [...rows]
  sorted.sort((left, right) => {
    const leftFail = compensationFailCount(left, version)
    const rightFail = compensationFailCount(right, version)
    if (leftFail !== rightFail) {
      return rightFail - leftFail
    }
    if (left.channel_id !== right.channel_id) {
      return left.channel_id - right.channel_id
    }
    return left.model_name.localeCompare(right.model_name)
  })
  return sorted
}

export function countCompensatedSuppliers(
  suppliers: AvailabilityCompensationSupplierRow[],
  version: AvailabilityCompensationVersion
): number {
  return suppliers.filter((row) =>
    version === 'all'
      ? row.compensation_pct_all > 0
      : row.compensation_pct_service > 0
  ).length
}

export function groupCompensationRowsBySupplier(
  rows: AvailabilityCompensationRow[],
  suppliers: AvailabilityCompensationSupplierRow[],
  version: AvailabilityCompensationVersion
): CompensationRowGroup[] {
  const bySupplier = new Map<string, AvailabilityCompensationRow[]>()
  for (const row of rows) {
    const group = bySupplier.get(row.supplier)
    if (group) {
      group.push(row)
    } else {
      bySupplier.set(row.supplier, [row])
    }
  }
  const sortedSuppliers = [...suppliers].sort((left, right) => {
    const leftFail = compensationFailCount(left, version)
    const rightFail = compensationFailCount(right, version)
    if (leftFail !== rightFail) {
      return rightFail - leftFail
    }
    return left.supplier.localeCompare(right.supplier)
  })
  return sortedSuppliers.map((supplier) => ({
    supplier,
    rows: sortCompensationRows(bySupplier.get(supplier.supplier) ?? [], version),
  }))
}
