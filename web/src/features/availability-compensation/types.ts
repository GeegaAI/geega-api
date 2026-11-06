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
export type AvailabilityCompensationTier =
  | 'none'
  | 'tier_99'
  | 'tier_95'
  | 'below_95'

export type AvailabilityCompensationVersion = 'all' | 'service'

export interface AvailabilityCompensationRow {
  model_name: string
  channel_id: number
  channel_name: string
  supplier: string
  ok_requests: number
  fail_all_requests: number
  fail_service_requests: number
  error_rate_all: number
  error_rate_service: number
  error_minutes_ge50_all: number
  error_minutes_any_all: number
  error_minutes_ge50_service: number
  error_minutes_any_service: number
  availability_ge50_all: number
  availability_any_all: number
  availability_ge50_service: number
  availability_any_service: number
  tier_all: AvailabilityCompensationTier
  tier_service: AvailabilityCompensationTier
  compensation_pct_all: number
  compensation_pct_service: number
}

export interface AvailabilityCompensationSupplierRow {
  supplier: string
  channel_count: number
  ok_requests: number
  fail_all_requests: number
  fail_service_requests: number
  error_rate_all: number
  error_rate_service: number
  error_minutes_ge50_all: number
  error_minutes_any_all: number
  error_minutes_ge50_service: number
  error_minutes_any_service: number
  availability_ge50_all: number
  availability_any_all: number
  availability_ge50_service: number
  availability_any_service: number
  tier_all: AvailabilityCompensationTier
  tier_service: AvailabilityCompensationTier
  compensation_pct_all: number
  compensation_pct_service: number
}

export interface AvailabilityCompensationData {
  start_timestamp: number
  end_timestamp: number
  total_minutes: number
  snapshot: boolean
  data_start_timestamp: number
  data_end_timestamp: number
  channel_count: number
  model_count: number
  supplier_count: number
  rows: AvailabilityCompensationRow[]
  suppliers: AvailabilityCompensationSupplierRow[]
}

export interface AvailabilityCompensationResponse {
  success: boolean
  message?: string
  data?: AvailabilityCompensationData
}
