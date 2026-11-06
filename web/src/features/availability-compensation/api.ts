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

import type { AvailabilityCompensationResponse } from './types'

interface AvailabilityCompensationParams {
  startTimestamp: number
  endTimestamp: number
}

function rangeParams(params: AvailabilityCompensationParams) {
  return {
    start_timestamp: params.startTimestamp,
    end_timestamp: params.endTimestamp,
  }
}

export async function getAvailabilityCompensation(
  params: AvailabilityCompensationParams
): Promise<AvailabilityCompensationResponse> {
  const res = await api.get<AvailabilityCompensationResponse>(
    '/api/log/availability-compensation',
    { params: rangeParams(params) }
  )
  if (!res.data.success) {
    throw new Error(res.data.message || 'Failed to load')
  }
  return res.data
}

export async function downloadAvailabilityCompensation(
  params: AvailabilityCompensationParams,
  filename: string
): Promise<void> {
  const res = await api.get('/api/log/availability-compensation/export', {
    params: rangeParams(params),
    responseType: 'blob',
  })
  const blob = res.data as Blob
  if (blob.type && !blob.type.includes('spreadsheetml')) {
    throw new Error('Failed to export Excel')
  }
  const url = URL.createObjectURL(
    new Blob([blob], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    })
  )
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}
