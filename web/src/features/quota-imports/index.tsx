import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useIsFetching, useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import { QuotaRecordsCard } from './components/quota-records-card'
import { QuotaUploadCard } from './components/quota-upload-card'

export function QuotaImports() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const fetching = useIsFetching({ queryKey: ['quota-import-records'] })

  const handleUploaded = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ['quota-import-records'] })
  }, [queryClient])

  const handleRefresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ['quota-import-records'] })
    void queryClient.invalidateQueries({
      queryKey: ['quota-import-record-detail'],
    })
  }, [queryClient])

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Monthly Quota Setting')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={handleRefresh}
          disabled={fetching > 0}
          aria-label={t('Refresh')}
        >
          <RefreshCw
            data-icon='inline-start'
            className={cn('size-3.5', fetching > 0 && 'animate-spin')}
            aria-hidden='true'
          />
          {fetching > 0 ? t('Refreshing...') : t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <QuotaUploadCard onUploaded={handleUploaded} />
          <QuotaRecordsCard />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
