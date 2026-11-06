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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import {
  getLegalVersions,
  publishLegalVersion,
} from '@/features/legal/api'
import { invalidateLegalStatusCache } from '../hooks/use-update-option'

const PUBLISH_HISTORY_LIMIT = 5

// 法律文档版本面板：编辑保存不产生新版本，只有点击“发布新版本”才
// 固化快照并要求所有用户重新阅读确认。
export function LegalVersionsPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['legal-versions'],
    queryFn: getLegalVersions,
  })
  const versionsData = data?.data

  const publishMutation = useMutation({
    mutationFn: publishLegalVersion,
    onSuccess: async (res) => {
      toast.success(
        t('Version {{version}} published', {
          version: res?.data?.version ?? '',
        })
      )
      queryClient.invalidateQueries({ queryKey: ['legal-versions'] })
      await invalidateLegalStatusCache(queryClient)
      setConfirmOpen(false)
    },
  })

  const currentVersion = versionsData?.current_version ?? 0
  const nextVersion = currentVersion + 1
  const history = (versionsData?.versions ?? []).slice(
    0,
    PUBLISH_HISTORY_LIMIT
  )

  const formatPublishedAt = (seconds: number) =>
    seconds > 0 ? dayjs(seconds * 1000).format('YYYY-MM-DD HH:mm') : ''

  const versionBadge = (() => {
    if (isLoading) return <Spinner className='h-3 w-3' />
    if (currentVersion > 0) return `v${currentVersion}`
    return t('Not published yet')
  })()

  return (
    <div className='rounded-lg border p-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='space-y-1'>
          <div className='flex items-center gap-2'>
            <span className='text-foreground text-sm font-medium'>
              {t('Current version')}
            </span>
            <span
              className='bg-primary/10 text-primary rounded-full px-2 py-0.5 text-xs font-semibold'
              data-testid='legal-current-version'
            >
              {versionBadge}
            </span>
          </div>
        </div>
        <Button
          type='button'
          size='sm'
          disabled={publishMutation.isPending}
          onClick={() => setConfirmOpen(true)}
        >
          {publishMutation.isPending ? <Spinner className='h-4 w-4' /> : null}
          {t('Publish new version')}
        </Button>
      </div>

      {history.length > 0 && (
        <div className='mt-3 border-t pt-3'>
          <p className='text-muted-foreground text-xs font-medium'>
            {t('Version history')}
          </p>
          <ul className='mt-1.5 space-y-1'>
            {history.map((item) => (
              <li
                key={item.version}
                className='text-muted-foreground text-xs leading-5'
              >
                v{item.version} · {formatPublishedAt(item.published_at)}
                {item.publisher_username
                  ? ` · ${t('Published by {{publisher}}', {
                      publisher: item.publisher_username,
                    })}`
                  : ''}
              </li>
            ))}
          </ul>
        </div>
      )}

      <Dialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('Publish new version?')}
        description={t(
          'Publishing will snapshot the current content of all documents as version {{version}}. All users will be required to read and confirm again.',
          { version: nextVersion }
        )}
        contentClassName='max-w-md'
        headerClassName='text-left'
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <>
            <Button
              type='button'
              variant='outline'
              onClick={() => setConfirmOpen(false)}
              disabled={publishMutation.isPending}
            >
              {t('Cancel')}
            </Button>
            <Button
              type='button'
              onClick={() => publishMutation.mutate()}
              disabled={publishMutation.isPending}
              className='gap-2'
            >
              {publishMutation.isPending ? (
                <Spinner className='h-4 w-4' />
              ) : null}
              {t('Publish new version')}
            </Button>
          </>
        }
      >
        <span className='hidden' aria-hidden='true' />
      </Dialog>
    </div>
  )
}
