import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { downloadQuotaImportTemplate, previewQuotaImport, uploadQuotaImport } from '../api'
import type { QuotaImportPreviewSummary } from '../types'

interface QuotaUploadCardProps {
  onUploaded: () => void
}

export function QuotaUploadCard(props: QuotaUploadCardProps) {
  const { t } = useTranslation()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<QuotaImportPreviewSummary | null>(null)
  const [previewing, setPreviewing] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [uploading, setUploading] = useState(false)

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const selected = event.target.files?.[0] ?? null
    setFile(selected)
    setPreview(null)
  }

  const handlePreview = async () => {
    if (!file) return
    setPreviewing(true)
    try {
      const summary = await previewQuotaImport(file)
      setPreview(summary)
    } catch {
      toast.error(t('Failed to parse Excel file'))
    } finally {
      setPreviewing(false)
    }
  }

  const handleDownloadTemplate = async () => {
    try {
      await downloadQuotaImportTemplate()
    } catch {
      toast.error(t('Failed to download template'))
    }
  }

  const handleUpload = async () => {
    if (!file || !preview) return
    setUploading(true)
    try {
      const record = await uploadQuotaImport(file, preview.content_hash)
      toast.success(
        t('Task created: {{filename}} ({{count}} rows)', {
          filename: record.filename,
          count: record.total_rows,
        })
      )
      setPreview(null)
      setFile(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
      props.onUploaded()
    } catch {
      toast.error(t('Failed to create import task'))
    } finally {
      setUploading(false)
      setConfirmOpen(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Upload Quota Excel')}</CardTitle>
      </CardHeader>
      <CardContent className='space-y-4'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Uploading creates a pending task only. Execute it manually or schedule a time (e.g. night) from the task list. Executing applies the quotas in the file to users on the monthly fixed-day policy; users on cycle or no-reset policies are skipped.'
          )}
        </p>
        <div className='flex flex-wrap items-center gap-3'>
          <input
            ref={fileInputRef}
            id='quota-import-file'
            type='file'
            accept='.xlsx'
            onChange={handleFileChange}
            className='hidden'
          />
          <label
            htmlFor='quota-import-file'
            className='inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-lg border border-primary bg-background px-2.5 text-sm font-medium text-primary transition-colors hover:bg-primary/5'
          >
            {t('Select File')}
          </label>
          {file && (
            <span
              className='text-muted-foreground max-w-60 truncate text-sm'
              title={file.name}
            >
              {file.name}
            </span>
          )}
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={handleDownloadTemplate}
          >
            {t('Download Template')}
          </Button>
          <Button onClick={handlePreview} disabled={!file || previewing}>
            {previewing ? t('Parsing...') : t('Preview')}
          </Button>
          <Button
            onClick={() => setConfirmOpen(true)}
            disabled={!preview || uploading}
          >
            {uploading ? t('Uploading...') : t('Create Task')}
          </Button>
        </div>

        {preview && (
          <div className='space-y-2'>
            <div className='flex flex-wrap items-center gap-2 text-sm'>
              <Badge variant='outline'>
                {t('{{count}} rows', { count: preview.total })}
              </Badge>
              <Badge
                variant={preview.unmatched > 0 ? 'destructive' : 'outline'}
              >
                {t('{{count}} unmatched', { count: preview.unmatched })}
              </Badge>
              {preview.issue > 0 && (
                <Badge variant='secondary'>
                  {t('{{count}} with issues', { count: preview.issue })}
                </Badge>
              )}
            </div>
            <div className='max-h-72 overflow-auto rounded-md border'>
              <table className='w-full border-collapse text-sm'>
                <thead className='bg-muted/50 sticky top-0'>
                  <tr>
                    <th className='border border-border p-2 text-start'>{t('Row')}</th>
                    <th className='border border-border p-2 text-start'>{t('Account')}</th>
                    <th className='border border-border p-2 text-end'>{t('Amount (CNY)')}</th>
                    <th className='border border-border p-2 text-start'>{t('Status')}</th>
                  </tr>
                </thead>
                <tbody>
                  {preview.rows.map((row) => (
                    <tr key={row.row_number} className='border-t'>
                      <td className='border border-border p-2'>{row.row_number}</td>
                      <td className='border border-border p-2 font-mono'>{row.username}</td>
                      <td className='border border-border p-2 text-end'>{row.amount_yuan}</td>
                      <td className='border border-border p-2'>
                        {row.issue && (
                          <Badge variant='destructive'>{row.issue}</Badge>
                        )}
                        {!row.issue && row.matched && (
                          <Badge variant='outline'>{t('OK')}</Badge>
                        )}
                        {!row.issue && !row.matched && (
                          <Badge variant='destructive'>
                            {t('Not Found')}
                          </Badge>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        <ConfirmDialog
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          title={t('Create import task')}
          desc={t(
            'A pending task will be created without executing. You can execute or schedule it from the task list. Continue?'
          )}
          confirmText={t('Create Task')}
          isLoading={uploading}
          handleConfirm={handleUpload}
        />
      </CardContent>
    </Card>
  )
}
