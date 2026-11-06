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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { getPricing } from '@/features/pricing/api'
import { cn } from '@/lib/utils'

import { getUserModelRatios, setUserModelRatios } from '../api'
import type { UserModelRatioMap } from '../types'
import {
  BILLING_MODE_TIERED_EXPR,
  QUOTA_TYPE_PER_REQUEST,
  buildModelRatioRows,
  buildRatiosPayload,
  parseRatioDraft,
  type UserModelRatioRow,
} from '../lib'

const PAGE_SIZE = 50

/** Billing-type badge for a row: per-request, tiered, or none */
function billingBadgeFor(row: UserModelRatioRow): {
  label: string
  variant: 'outline'
} | null {
  if (row.quotaType === QUOTA_TYPE_PER_REQUEST) {
    return { label: 'Per-request', variant: 'outline' }
  }
  if (row.billingMode === BILLING_MODE_TIERED_EXPR) {
    return { label: 'Tiered', variant: 'outline' }
  }
  return null
}

interface UserModelRatioEditorProps {
  userId: number
  /** Target user's group; only models enabled for it are listed */
  group: string
  /** Called after the overrides were saved successfully */
  onSaved?: () => void
}

/**
 * Editor for per-user model discounts. Rows left empty mean "no discount"
 * (billed at the global price); 0 makes the model free for this user. Saving performs
 * a full replacement (PUT) of the override map.
 */
export function UserModelRatioEditor(props: UserModelRatioEditorProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [onlyOverridden, setOnlyOverridden] = useState(false)
  const [page, setPage] = useState(1)
  const [draft, setDraft] = useState<Record<string, string>>({})

  const ratiosQuery = useQuery({
    queryKey: ['user-model-ratios', props.userId],
    queryFn: () => getUserModelRatios(props.userId),
    staleTime: 5 * 60 * 1000,
  })

  const pricingQuery = useQuery({
    queryKey: ['pricing'],
    queryFn: getPricing,
    staleTime: 5 * 60 * 1000,
  })

  const overrides: UserModelRatioMap = useMemo(
    () => ratiosQuery.data?.data ?? {},
    [ratiosQuery.data]
  )

  // Seed the draft inputs from the fetched overrides (empty input = inherit)
  useEffect(() => {
    if (!ratiosQuery.isSuccess) return
    const initial: Record<string, string> = {}
    for (const [modelName, ratio] of Object.entries(overrides)) {
      initial[modelName] = String(ratio)
    }
    setDraft(initial)
  }, [ratiosQuery.isSuccess, overrides])

  const rows = useMemo(() => {
    const models = pricingQuery.data?.data ?? []
    return buildModelRatioRows(models, props.group, overrides)
  }, [pricingQuery.data, props.group, overrides])

  const filteredRows = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    return rows.filter((row) => {
      if (keyword && !row.modelName.toLowerCase().includes(keyword)) {
        return false
      }
      if (onlyOverridden && !row.overridden) return false
      return true
    })
  }, [rows, search, onlyOverridden])

  const totalPages = Math.max(1, Math.ceil(filteredRows.length / PAGE_SIZE))
  const currentPage = Math.min(page, totalPages)
  const pageRows = filteredRows.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE
  )

  const getEmptyStateMessage = (): string | null => {
    if (rows.length === 0) return t('No models available for this group.')
    if (filteredRows.length === 0) return t('No matching models.')
    return null
  }

  const emptyMessage = getEmptyStateMessage()

  const hasInvalidDraft = useMemo(
    () =>
      Object.values(draft).some(
        (value) => parseRatioDraft(value).kind === 'invalid'
      ),
    [draft]
  )

  const saveMutation = useMutation({
    mutationFn: (ratios: UserModelRatioMap) =>
      setUserModelRatios(props.userId, ratios),
    onSuccess: (result, ratios) => {
      if (result.success) {
        queryClient.setQueryData(['user-model-ratios', props.userId], {
          success: true,
          message: result.message,
          data: ratios,
        })
        toast.success(t('Model ratio overrides saved'))
        props.onSaved?.()
      } else {
        toast.error(
          result.message || t('Failed to save model ratio overrides')
        )
      }
    },
    onError: (error: unknown) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to save model ratio overrides')
      )
    },
  })

  const handleSave = () => {
    const payload = buildRatiosPayload(draft)
    if (payload === null) {
      toast.error(
        t('Discount must be a number greater than or equal to 0 and less than 1')
      )
      return
    }
    saveMutation.mutate(payload)
  }

  const updateDraft = (modelName: string, value: string) => {
    setDraft((prev) => ({ ...prev, [modelName]: value }))
  }

  const handleRetry = () => {
    ratiosQuery.refetch()
    pricingQuery.refetch()
  }

  if (ratiosQuery.isPending || pricingQuery.isPending) {
    return (
      <div className='flex items-center gap-2 py-8 text-sm text-muted-foreground'>
        <Spinner />
        {t('Loading model ratios...')}
      </div>
    )
  }

  if (ratiosQuery.isError || pricingQuery.isError) {
    const message = ratiosQuery.isError
      ? t('Failed to load model ratios')
      : t('Failed to load pricing data')
    return (
      <div className='flex flex-col items-start gap-2 py-6'>
        <p className='text-destructive text-sm'>{message}</p>
        <Button variant='outline' size='sm' onClick={handleRetry}>
          {t('Retry')}
        </Button>
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      <p className='text-muted-foreground text-xs'>
        {t(
          'Leave empty to inherit the global price. Set 0 to make this model free for this user.'
        )}
        {' '}
        {t(
          'For tiered and per-request models, the custom ratio acts as a coefficient applied on top of the global price.'
        )}
      </p>

      <div className='flex flex-wrap items-center gap-2'>
        <Input
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(1)
          }}
          placeholder={t('Search models...')}
          aria-label={t('Search models...')}
          className='h-8 max-w-64 flex-1'
        />
        <label className='flex items-center gap-1.5 text-sm'>
          <Switch
            size='sm'
            checked={onlyOverridden}
            onCheckedChange={(checked) => {
              setOnlyOverridden(checked)
              setPage(1)
            }}
          />
          {t('Only show overridden')}
        </label>
        <span className='text-muted-foreground ml-auto text-xs'>
          {t('Page {{page}} of {{total}}', {
            page: currentPage,
            total: totalPages,
          })}
        </span>
      </div>

      {emptyMessage ? (
        <p className='py-6 text-center text-sm text-muted-foreground'>
          {emptyMessage}
        </p>
      ) : (
        <>
          <div className='grid grid-cols-[minmax(0,1fr)_5rem_9rem] gap-2 border-b pb-1 text-xs font-medium text-muted-foreground'>
            <span>{t('Model')}</span>
            <span className='text-right'>{t('Global value')}</span>
            <span>{t('Custom ratio')}</span>
          </div>
          <div className='max-h-[340px] space-y-1 overflow-y-auto pr-1'>
            {pageRows.map((row) => {
              const draftValue = draft[row.modelName] ?? ''
              const invalid = parseRatioDraft(draftValue).kind === 'invalid'
              const isPerRequest = row.quotaType === QUOTA_TYPE_PER_REQUEST
              const billingBadge = billingBadgeFor(row)
              let globalValueLabel: string
              if (isPerRequest) {
                globalValueLabel = `$${row.modelPrice} / ${t('request')}`
              } else if (row.configured) {
                globalValueLabel = String(row.globalRatio)
              } else {
                globalValueLabel = t('Not configured')
              }
              return (
                <div
                  key={row.modelName}
                  className='grid grid-cols-[minmax(0,1fr)_5rem_9rem] items-center gap-2'
                >
                  <div className='flex min-w-0 items-center gap-1.5'>
                    <span className='truncate text-sm' title={row.modelName}>
                      {row.modelName}
                    </span>
                    {row.overridden ? (
                      <Badge variant='secondary'>{t('Overridden')}</Badge>
                    ) : null}
                    {billingBadge ? (
                      <Badge variant={billingBadge.variant}>
                        {t(billingBadge.label)}
                      </Badge>
                    ) : null}
                  </div>
                  <span className='text-right text-sm text-muted-foreground'>
                    {globalValueLabel}
                  </span>
                  <Input
                    type='number'
                    min={0}
                    step='any'
                    aria-label={t('Custom ratio: {{model}}', {
                      model: row.modelName,
                    })}
                    placeholder={
                      isPerRequest
                        ? t('Inherit global price')
                        : t('Inherit global ratio')
                    }
                    aria-invalid={invalid}
                    className={cn('h-8', invalid && 'border-destructive')}
                    value={draftValue}
                    onChange={(e) => updateDraft(row.modelName, e.target.value)}
                  />
                </div>
              )
            })}
          </div>
          <div className='flex items-center justify-end gap-2'>
            <Button
              variant='outline'
              size='sm'
              disabled={currentPage <= 1}
              onClick={() => setPage(currentPage - 1)}
            >
              {t('Previous page')}
            </Button>
            <Button
              variant='outline'
              size='sm'
              disabled={currentPage >= totalPages}
              onClick={() => setPage(currentPage + 1)}
            >
              {t('Next page')}
            </Button>
          </div>
        </>
      )}

      {hasInvalidDraft ? (
        <p className='text-destructive text-xs'>
          {t(
            'Discount must be a number greater than or equal to 0 and less than 1'
          )}
        </p>
      ) : null}

      <div className='flex justify-end'>
        <Button
          size='sm'
          onClick={handleSave}
          disabled={saveMutation.isPending}
        >
          {saveMutation.isPending ? t('Saving...') : t('Save')}
        </Button>
      </div>
    </div>
  )
}
