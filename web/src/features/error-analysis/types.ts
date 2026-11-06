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
export interface ErrorAnalysisSummary {
  error_count: number
  request_count: number
  error_rate: number
  affected_users: number
  affected_channels: number
  affected_models: number
}

export interface ErrorTrendPoint {
  timestamp: number
  error_count: number
}

export interface ErrorReasonStat {
  code: string
  count: number
  ratio: number
  status_codes: number[]
  channel_count: number
  sample_message: string
}

export interface ErrorChannelStat {
  channel_id: number
  channel_name: string
  channel_type: number
  error_count: number
  request_count: number
  error_rate: number
  top_reason_code: string
  top_reason_count: number
  model_count: number
  user_count: number
  last_error_at: number
}

export interface ErrorModelStat {
  model_name: string
  error_count: number
  request_count: number
  error_rate: number
  top_reason_code: string
  top_reason_count: number
  channel_count: number
  user_count: number
  last_error_at: number
}

export interface ErrorBreakdownRow {
  channel_id: number
  channel_name: string
  model_name: string
  reason_code: string
  error_count: number
  reason_ratio: number
  channel_model_request_count: number
  channel_model_error_count: number
  channel_model_error_rate: number
  status_codes: number[]
  user_count: number
  sample_message: string
}

export interface ErrorModelDetailRow {
  model_name: string
  error_count: number
  request_count: number
  error_rate: number
  top_reason_code: string
  status_codes: number[]
  sample_message: string
}

export interface ErrorChannelGroupRow {
  channel: ErrorChannelStat
  models: ErrorModelDetailRow[]
}

export interface ErrorProviderGroup {
  key: string
  channel_type: number
  type_label: string
  known_type: boolean
  channels: ErrorChannelGroupRow[]
  error_count: number
  request_count: number
  error_rate: number
  model_count: number
  top_reason_code: string
  status_codes: number[]
  last_error_at: number
}

export interface ErrorAnalysisData {
  start_timestamp: number
  end_timestamp: number
  bucket: 'hour' | 'day'
  truncated: boolean
  rows_truncated: boolean
  summary: ErrorAnalysisSummary
  trend: ErrorTrendPoint[]
  reasons: ErrorReasonStat[]
  channels: ErrorChannelStat[]
  models: ErrorModelStat[]
  rows: ErrorBreakdownRow[]
}

export interface ErrorAnalysisResponse {
  success: boolean
  message?: string
  data?: ErrorAnalysisData
}
