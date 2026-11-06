import { api } from '@/lib/api'

import type {
  ApiResponse,
  QuotaImportDetail,
  QuotaImportPreviewSummary,
  QuotaImportRecord,
} from './types'

export function quotaImportTemplateUrl(): string {
  return '/api/quota_import/template'
}

export async function downloadQuotaImportTemplate(): Promise<void> {
  const res = await api.get('/api/quota_import/template', {
    responseType: 'blob',
  })
  const url = URL.createObjectURL(
    new Blob([res.data as Blob], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    })
  )
  const a = document.createElement('a')
  a.href = url
  a.download = '额度申请模版.xlsx'
  a.click()
  URL.revokeObjectURL(url)
}

export async function previewQuotaImport(
  file: File
): Promise<QuotaImportPreviewSummary> {
  const form = new FormData()
  form.append('file', file)
  const res = await api.post<ApiResponse<QuotaImportPreviewSummary>>(
    '/api/quota_import/preview',
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } }
  )
  return res.data.data
}

export async function uploadQuotaImport(
  file: File,
  contentHash: string,
  scheduledTime = 0
): Promise<QuotaImportRecord> {
  const form = new FormData()
  form.append('file', file)
  form.append('content_hash', contentHash)
  form.append('scheduled_time', String(scheduledTime))
  const res = await api.post<ApiResponse<QuotaImportRecord>>(
    '/api/quota_import/upload',
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } }
  )
  return res.data.data
}

export async function executeQuotaImportRecord(
  id: number
): Promise<QuotaImportRecord> {
  const res = await api.post<ApiResponse<QuotaImportRecord>>(
    `/api/quota_import/records/${id}/execute`
  )
  return res.data.data
}

export async function scheduleQuotaImportRecord(
  id: number,
  scheduledTime: number
): Promise<void> {
  await api.put(`/api/quota_import/records/${id}/schedule`, {
    scheduled_time: scheduledTime,
  })
}

export async function deleteQuotaImportRecord(id: number): Promise<void> {
  await api.delete(`/api/quota_import/records/${id}`)
}

export async function getQuotaImportRecords(params: {
  p?: number
  page_size?: number
}): Promise<{
  records: QuotaImportRecord[]
  total: number
  page: number
  page_size: number
}> {
  const res = await api.get<ApiResponse<{
    records: QuotaImportRecord[]
    total: number
    page: number
    page_size: number
  }>>('/api/quota_import/records', { params })
  return res.data.data
}

export async function getQuotaImportRecordDetail(
  id: number
): Promise<QuotaImportDetail[]> {
  const res = await api.get<ApiResponse<QuotaImportDetail[]>>(
    `/api/quota_import/records/${id}`
  )
  return res.data.data
}
