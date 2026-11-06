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
import { api } from '@/lib/api'

import type { ErrorAnalysisResponse } from './types'

interface ErrorAnalysisParams {
  startTimestamp: number
  endTimestamp: number
  channelId?: number
  modelName?: string
  group?: string
}

export async function getErrorAnalysis(
  params: ErrorAnalysisParams
): Promise<ErrorAnalysisResponse> {
  const res = await api.get<ErrorAnalysisResponse>('/api/log/error-analysis', {
    params: {
      start_timestamp: params.startTimestamp,
      end_timestamp: params.endTimestamp,
      timezone_offset: new Date().getTimezoneOffset(),
      channel_id: params.channelId || undefined,
      model_name: params.modelName || undefined,
      group: params.group || undefined,
    },
  })
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to load')
  }
  return res.data
}
