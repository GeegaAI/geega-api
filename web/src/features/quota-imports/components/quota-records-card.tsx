import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import dayjs from 'dayjs'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { formatQuota } from '@/lib/format'
import {
  deleteQuotaImportRecord,
  executeQuotaImportRecord,
  getQuotaImportRecordDetail,
  getQuotaImportRecords,
  scheduleQuotaImportRecord,
} from '../api'
import type { QuotaImportDetail, QuotaImportRecord } from '../types'

const PAGE_SIZE = 20

function statusBadgeVariant(status: QuotaImportRecord['status']) {
  if (status === 'success') return 'outline' as const
  if (status === 'pending' || status === 'executing' || status === 'partial') {
    return 'secondary' as const
  }
  return 'destructive' as const
}

function formatDateTime(ts: number) {
  if (!ts) return '—'
  return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

// datetime-local 输入框值必须用本地时间构造；toISOString 是 UTC 会差时区
function toLocalInputValue(ts: number) {
  return dayjs(ts * 1000).format('YYYY-MM-DDTHH:mm')
}

function statusLabel(t: (key: string) => string, status: string) {
  if (status === 'pending') return t('Pending')
  if (status === 'executing') return t('Executing')
  if (status === 'partial') return t('Partial')
  if (status === 'failed') return t('Failed')
  return t('Success')
}

export function QuotaRecordsCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [detailId, setDetailId] = useState<number | null>(null)
  const [executeTarget, setExecuteTarget] = useState<QuotaImportRecord | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<QuotaImportRecord | null>(null)
  const [scheduleTarget, setScheduleTarget] = useState<QuotaImportRecord | null>(null)
  const [scheduleValue, setScheduleValue] = useState('')
  const [busy, setBusy] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['quota-import-records', page],
    queryFn: () => getQuotaImportRecords({ p: page, page_size: PAGE_SIZE }),
  })

  const { data: details, isLoading: detailsLoading } = useQuery({
    queryKey: ['quota-import-record-detail', detailId],
    queryFn: () => getQuotaImportRecordDetail(detailId as number),
    enabled: detailId !== null,
  })

  const records = data?.records ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['quota-import-records'] })
    void queryClient.invalidateQueries({ queryKey: ['quota-import-record-detail'] })
  }

  const handleExecuteNow = async () => {
    if (!executeTarget) return
    setBusy(true)
    try {
      const record = await executeQuotaImportRecord(executeTarget.id)
      toast.success(
        t('Task executed: {{status}}, {{success}} success', {
          status: record.status,
          success: record.success_rows,
        })
      )
      refresh()
    } catch {
      toast.error(t('Failed to execute task'))
    } finally {
      setBusy(false)
      setExecuteTarget(null)
    }
  }

  const handleScheduleSave = async () => {
    if (!scheduleTarget) return
    setBusy(true)
    try {
      const ts = scheduleValue
        ? Math.floor(new Date(scheduleValue).getTime() / 1000)
        : 0
      if (scheduleValue && Number.isNaN(ts)) {
        toast.error(t('Invalid schedule time'))
        return
      }
      await scheduleQuotaImportRecord(scheduleTarget.id, ts)
      toast.success(scheduleValue ? t('Schedule saved') : t('Schedule cleared'))
      setScheduleTarget(null)
      refresh()
    } catch {
      toast.error(t('Failed to save schedule'))
    } finally {
      setBusy(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setBusy(true)
    try {
      await deleteQuotaImportRecord(deleteTarget.id)
      toast.success(t('Record deleted'))
      refresh()
    } catch {
      toast.error(t('Failed to delete record'))
    } finally {
      setBusy(false)
      setDeleteTarget(null)
    }
  }

  const openScheduleDialog = (record: QuotaImportRecord) => {
    setScheduleTarget(record)
    setScheduleValue(
      record.scheduled_time ? toLocalInputValue(record.scheduled_time) : ''
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Execution Records')}</CardTitle>
      </CardHeader>
      <CardContent className='space-y-3'>
        {isLoading && (
          <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
        )}
        {!isLoading && records.length === 0 && (
          <p className='text-muted-foreground text-sm'>{t('No records yet')}</p>
        )}
        {!isLoading && records.length > 0 && (
          <div className='max-h-96 overflow-auto rounded-md border'>
            <table className='w-full border-collapse text-sm'>
              <thead className='bg-muted/50 sticky top-0'>
                <tr>
                  <th className='border border-border p-2 text-start'>{t('Created At')}</th>
                  <th className='border border-border p-2 text-start'>{t('File')}</th>
                  <th className='border border-border p-2 text-start whitespace-nowrap min-w-24'>
                    {t('Operator')}
                  </th>
                  <th className='border border-border p-2 text-start'>{t('Scheduled Time')}</th>
                  <th className='border border-border p-2 text-start'>{t('Executed At')}</th>
                  <th className='border border-border p-2 text-end whitespace-nowrap min-w-16'>
                    {t('Reset')}
                  </th>
                  <th className='border border-border p-2 text-end whitespace-nowrap min-w-16'>
                    {t('Success')}
                  </th>
                  <th className='border border-border p-2 text-end whitespace-nowrap min-w-20'>
                    {t('Unmatched')}
                  </th>
                  <th className='border border-border p-2 text-start'>{t('Status')}</th>
                  <th className='border border-border p-2 text-start'>{t('Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {records.map((record) => (
                  <tr key={record.id} className='border-t'>
                    <td className='border border-border p-2 whitespace-nowrap'>
                      {formatDateTime(record.created_time)}
                    </td>
                    <td className='border border-border p-2 max-w-40 truncate' title={record.filename}>
                      {record.filename}
                    </td>
                    <td className='border border-border p-2 whitespace-nowrap min-w-24'>
                      {record.operator_name}
                    </td>
                    <td className='border border-border p-2 whitespace-nowrap'>
                      {formatDateTime(record.scheduled_time)}
                    </td>
                    <td className='border border-border p-2 whitespace-nowrap'>
                      {formatDateTime(record.executed_time)}
                    </td>
                    <td className='border border-border p-2 text-end whitespace-nowrap min-w-16'>
                      {record.type === 'upload' ? '—' : record.reset_count}
                    </td>
                    <td className='border border-border p-2 text-end whitespace-nowrap min-w-16'>
                      {record.success_rows}
                    </td>
                    <td className='border border-border p-2 text-end whitespace-nowrap min-w-20'>
                      {record.unmatched_rows + record.failed_rows}
                    </td>
                    <td className='border border-border p-2'>
                      <Badge variant={statusBadgeVariant(record.status)}>
                        {statusLabel(t, record.status)}
                      </Badge>
                    </td>
                    <td className='border border-border p-2'>
                      <div className='flex gap-1'>
                        {record.type === 'upload' && (
                          <Button
                            variant='ghost'
                            size='sm'
                            onClick={() => setDetailId(record.id)}
                          >
                            {t('Details')}
                          </Button>
                        )}
                        {record.type === 'upload' && record.status === 'pending' && (
                          <>
                            <Button
                              variant='ghost'
                              size='sm'
                              onClick={() => setExecuteTarget(record)}
                            >
                              {t('Execute Now')}
                            </Button>
                            <Button
                              variant='ghost'
                              size='sm'
                              onClick={() => openScheduleDialog(record)}
                            >
                              {t('Set Schedule')}
                            </Button>
                          </>
                        )}
                        <Button
                          variant='destructive'
                          size='sm'
                          onClick={() => setDeleteTarget(record)}
                        >
                          {t('Delete')}
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {totalPages > 1 && (
          <div className='flex items-center justify-end gap-2 text-sm'>
            <Button
              variant='outline'
              size='sm'
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              {t('Previous')}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant='outline'
              size='sm'
              disabled={page >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            >
              {t('Next')}
            </Button>
          </div>
        )}

        <Dialog
          open={detailId !== null}
          onOpenChange={(open) => !open && setDetailId(null)}
        >
          <DialogContent className='max-w-2xl'>
            <DialogHeader>
              <DialogTitle>{t('Import Details')}</DialogTitle>
            </DialogHeader>
            {detailsLoading ? (
              <p className='text-muted-foreground text-sm'>{t('Loading...')}</p>
            ) : (
              <div className='max-h-96 overflow-auto rounded-md border'>
                <table className='w-full border-collapse text-sm'>
                  <thead className='bg-muted/50 sticky top-0'>
                    <tr>
                      <th className='border border-border p-2 text-start'>{t('Account')}</th>
                      <th className='border border-border p-2 text-end'>{t('Amount (CNY)')}</th>
                      <th className='border border-border p-2 text-end'>{t('Quota')}</th>
                      <th className='border border-border p-2 text-start'>{t('Status')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(details ?? []).map((detail: QuotaImportDetail) => (
                      <tr key={detail.id} className='border-t'>
                        <td className='border border-border p-2 font-mono'>{detail.username}</td>
                        <td className='border border-border p-2 text-end'>{detail.amount_yuan}</td>
                        <td className='border border-border p-2 text-end'>
                          {detail.quota > 0 ? formatQuota(detail.quota) : '—'}
                        </td>
                        <td className='border border-border p-2'>
                          {detail.result === 'success' && (
                            <Badge variant='outline'>{t('OK')}</Badge>
                          )}
                          {detail.result === 'unmatched' && (
                            <Badge variant='destructive' title={detail.error_message}>
                              {t('Not Found')}
                            </Badge>
                          )}
                          {detail.result === 'skipped' && (
                            <Badge variant='secondary' title={detail.error_message}>
                              {t('Skipped (non-monthly policy)')}
                            </Badge>
                          )}
                          {detail.result === 'failed' && (
                            <Badge variant='destructive' title={detail.error_message}>
                              {t('Failed')}
                            </Badge>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </DialogContent>
        </Dialog>

        <ConfirmDialog
          open={executeTarget !== null}
          onOpenChange={(open) => !open && setExecuteTarget(null)}
          title={t('Execute this task now')}
          desc={t(
            'This applies the quotas from the task file to users on the monthly fixed-day policy. Users on cycle or no-reset policies are skipped. Continue?'
          )}
          confirmText={t('Execute')}
          isLoading={busy}
          handleConfirm={handleExecuteNow}
        />

        <ConfirmDialog
          open={deleteTarget !== null}
          onOpenChange={(open) => !open && setDeleteTarget(null)}
          title={t('Delete record')}
          desc={t('This record will be deleted permanently. Continue?')}
          confirmText={t('Delete')}
          destructive
          isLoading={busy}
          handleConfirm={handleDelete}
        />

        <Dialog
          open={scheduleTarget !== null}
          onOpenChange={(open) => !open && setScheduleTarget(null)}
        >
          <DialogContent className='sm:max-w-xl'>
            <DialogHeader>
              <DialogTitle>{t('Schedule execution time')}</DialogTitle>
            </DialogHeader>
            <div className='space-y-4'>
              <div className='space-y-2'>
                <label
                  htmlFor='quota-schedule-input'
                  className='text-sm font-medium'
                >
                  {t('Scheduled Time')}
                </label>
                <Input
                  id='quota-schedule-input'
                  type='datetime-local'
                  value={scheduleValue}
                  onChange={(e) => setScheduleValue(e.target.value)}
                  className='h-10 w-full'
                />
              </div>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Leave empty to clear the schedule; the task will only run manually.'
                )}
              </p>
              <div className='flex justify-end gap-2'>
                <Button
                  variant='outline'
                  onClick={() => setScheduleTarget(null)}
                  disabled={busy}
                >
                  {t('Cancel')}
                </Button>
                <Button onClick={handleScheduleSave} disabled={busy}>
                  {t('Save')}
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </CardContent>
    </Card>
  )
}
