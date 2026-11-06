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
import type { TFunction } from 'i18next'

import type { ErrorAnalysisRangePreset } from './lib/range'

export const ERROR_ANALYSIS_RANGE_PRESETS: {
  value: ErrorAnalysisRangePreset
  labelKey: string
}[] = [
  { value: 'yesterday', labelKey: 'Yesterday' },
  { value: 'today', labelKey: 'Today' },
  { value: 'last7days', labelKey: 'Last 7 Days' },
]

export const ERROR_REASON_LABEL_KEYS: Record<string, string> = {
  timeout: 'Upstream Timeout',
  upstream_unavailable: 'Upstream Unavailable',
  rate_limited: 'Upstream Rate Limited',
  quota_exhausted: 'Upstream Quota Exhausted',
  no_available_channel: 'No Available Channel',
  model_unavailable: 'Model Unavailable',
  auth_failed: 'Upstream Auth Failed',
  reasoning_effort_unsupported: 'Reasoning Effort Unsupported',
  endpoint_unsupported: 'Endpoint Unsupported',
  bad_request: 'Request Rejected by Upstream',
  content_filtered: 'Content Filtered',
  upstream_error: 'Upstream Server Error',
  other: 'Other Upstream Error',
}

export function getErrorReasonLabel(code: string, t: TFunction): string {
  const key = ERROR_REASON_LABEL_KEYS[code]
  return key ? t(key) : t('Other Upstream Error')
}
