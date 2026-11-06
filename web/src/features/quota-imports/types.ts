export interface QuotaImportPreviewRow {
  row_number: number
  username: string
  amount_yuan: number
  quota: number
  user_id: number
  matched: boolean
  current_quota: number
  non_monthly: boolean
  duplicate: boolean
  issue: string
}

export interface QuotaImportPreviewSummary {
  rows: QuotaImportPreviewRow[]
  total: number
  matched: number
  unmatched: number
  non_monthly: number
  issue: number
  total_amount: number
  filename: string
  content_hash: string
}

export interface QuotaImportRecord {
  id: number
  type: 'upload' | 'auto_reset'
  filename: string
  operator_id: number
  operator_name: string
  status: 'pending' | 'executing' | 'success' | 'partial' | 'failed'
  total_rows: number
  success_rows: number
  unmatched_rows: number
  skipped_rows: number
  failed_rows: number
  reset_count: number
  scheduled_time: number
  remark: string
  created_time: number
  executed_time: number
}

export interface QuotaImportDetail {
  id: number
  record_id: number
  username: string
  amount_yuan: number
  quota: number
  user_id: number
  result: 'success' | 'unmatched' | 'skipped' | 'failed'
  error_message: string
  created_time: number
}

export interface ExpiringCycleUser {
  id: number
  username: string
  display_name: string
  quota: number
  quota_next_reset_at: number
}

export interface ExpiringCycleLists {
  pending: ExpiringCycleUser[]
  expiring: ExpiringCycleUser[]
  days: number
}

export interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
}
