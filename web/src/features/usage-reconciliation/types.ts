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
export interface UsageReconciliationRow {
  usage_billing_path: string
  semantic: string
  channel_id: number
  channel_name: string
  model_name: string
  estimated: boolean
  local: boolean
  reconcilable: boolean
  request_count: number
  quota: number
  system_input_tokens: number
  system_output_tokens: number
  upstream_input_tokens: number
  reconciled_request_count?: number
}

export interface UsageReconciliationData {
  start_timestamp: number
  end_timestamp: number
  total: UsageReconciliationRow
  groups: UsageReconciliationRow[]
}

export interface UsageReconciliationResponse {
  success: boolean
  message?: string
  data?: UsageReconciliationData
}
