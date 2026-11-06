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
import { useQuery } from '@tanstack/react-query'
import { ChevronDown, ChevronRight, Code2, Eye, Plus, Save, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { JsonCodeEditor } from '@/components/json-code-editor'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { getEnabledModels } from '@/features/channels/api'

import { useUpdateOption } from '../hooks/use-update-option'
import {
  VIDEO_TIER_RESOLUTIONS,
  buildVideoTierPriceJson,
  parseVideoTierPriceJson,
  videoTierKey,
  type VideoTierKey,
  type VideoTierPriceEntry,
} from './video-tier-price'

type VideoTierPriceFormProps = {
  defaultValue: string
}

type EditorEntry = VideoTierPriceEntry & { id: string }

const MODEL_LIST_ID = 'video-tier-price-model-options'

// crypto.randomUUID 仅在安全上下文（HTTPS/localhost）可用，IP + HTTP 访问时需回退。
function createEditorId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `tier-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

function toEditorEntries(entries: VideoTierPriceEntry[]): EditorEntry[] {
  return entries.map((entry) => ({ ...entry, id: createEditorId() }))
}

function countConfiguredTiers(entry: VideoTierPriceEntry): number {
  return Object.values(entry.prices).filter(
    (value) => (value ?? '').trim() !== ''
  ).length
}

export function VideoTierPriceForm(props: VideoTierPriceFormProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [jsonMode, setJsonMode] = useState(false)
  const [entries, setEntries] = useState<EditorEntry[]>([])
  const [collapsedIds, setCollapsedIds] = useState<Set<string>>(new Set())
  const [jsonValue, setJsonValue] = useState('')
  const [jsonError, setJsonError] = useState<string | null>(null)

  const candidatesQuery = useQuery({
    queryKey: ['enabled-models'],
    queryFn: getEnabledModels,
  })
  const candidateModels = useMemo(
    () => candidatesQuery.data?.data ?? [],
    [candidatesQuery.data]
  )

  useEffect(() => {
    const parsed = parseVideoTierPriceJson(props.defaultValue)
    if (!parsed.ok) {
      setEntries([])
      setCollapsedIds(new Set())
      setJsonValue(props.defaultValue)
      setJsonError(null)
      return
    }
    const editorEntries = toEditorEntries(parsed.entries)
    setEntries(editorEntries)
    const built = buildVideoTierPriceJson(parsed.entries)
    setJsonValue(built.ok ? built.json : '{}')
    setJsonError(null)
    // 已配置价格的模型默认收起，便于浏览；新增模型保持展开。
    setCollapsedIds(
      new Set(
        editorEntries
          .filter((entry) => countConfiguredTiers(entry) > 0)
          .map((entry) => entry.id)
      )
    )
  }, [props.defaultValue])

  const addModel = useCallback(() => {
    setEntries((prev) => [
      ...prev,
      { id: createEditorId(), model: '', prices: {} },
    ])
  }, [])

  const removeModel = useCallback((index: number) => {
    setEntries((prev) => prev.filter((_, i) => i !== index))
  }, [])

  const updateModelName = useCallback((index: number, value: string) => {
    setEntries((prev) =>
      prev.map((entry, i) => (i === index ? { ...entry, model: value } : entry))
    )
  }, [])

  const updatePrice = useCallback(
    (index: number, key: VideoTierKey, value: string) => {
      setEntries((prev) =>
        prev.map((entry, i) => {
          if (i !== index) return entry
          return { ...entry, prices: { ...entry.prices, [key]: value } }
        })
      )
    },
    []
  )

  const toggleCollapsed = useCallback((id: string) => {
    setCollapsedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const toggleJsonMode = useCallback(() => {
    if (jsonMode) {
      const parsed = parseVideoTierPriceJson(jsonValue)
      if (!parsed.ok) {
        toast.error(t(parsed.error))
        return
      }
      setEntries(toEditorEntries(parsed.entries))
      setJsonError(null)
    } else {
      const built = buildVideoTierPriceJson(entries)
      if (!built.ok) {
        toast.error(t(built.error))
        return
      }
      setJsonValue(built.json)
    }
    setJsonMode((prev) => !prev)
  }, [entries, jsonMode, jsonValue, t])

  const handleJsonChange = useCallback((value: string) => {
    setJsonValue(value)
    if (!value.trim()) {
      setJsonError(null)
      return
    }
    const parsed = parseVideoTierPriceJson(value)
    setJsonError(parsed.ok ? null : parsed.error)
  }, [])

  const handleSave = useCallback(async () => {
    let payload: string
    if (jsonMode) {
      const parsed = parseVideoTierPriceJson(jsonValue)
      if (!parsed.ok) {
        setJsonError(parsed.error)
        toast.error(t(parsed.error))
        return
      }
      const built = buildVideoTierPriceJson(parsed.entries)
      if (!built.ok) {
        toast.error(t(built.error))
        return
      }
      payload = built.json
    } else {
      const built = buildVideoTierPriceJson(entries)
      if (!built.ok) {
        toast.error(t(built.error))
        return
      }
      payload = built.json
      setJsonValue(built.json)
    }

    await updateOption.mutateAsync({ key: 'VideoTierPrice', value: payload })
  }, [entries, jsonMode, jsonValue, t, updateOption])

  const isSaving = updateOption.isPending

  return (
    <div className='space-y-6'>
      <p className='text-sm leading-6 text-muted-foreground'>
        {t(
          'Prices are per model, resolution and video input, in CNY per 1M tokens; only their ratio to the base tier (720p, else 480p) affects billing, so absolute values do not change the charge. Empty cells fall back to the model ratio.'
        )}
      </p>
      <p className='text-xs leading-5 text-muted-foreground'>
        {t(
          'Allowed resolution keys: 480p, 720p, 768p, 1080p, 2k, 4k. Suffix _video marks requests with video input.'
        )}
      </p>

      <div className='flex flex-wrap justify-end gap-2'>
        <Button type='button' variant='outline' size='sm' onClick={addModel}>
          <Plus data-icon='inline-start' />
          {t('Add model')}
        </Button>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={toggleJsonMode}
        >
          {jsonMode ? (
            <>
              <Eye data-icon='inline-start' />
              {t('Switch to Visual')}
            </>
          ) : (
            <>
              <Code2 data-icon='inline-start' />
              {t('Switch to JSON')}
            </>
          )}
        </Button>
        <Button
          type='button'
          size='sm'
          onClick={handleSave}
          disabled={isSaving}
        >
          <Save data-icon='inline-start' />
          {isSaving ? t('Saving...') : t('Save video model pricing')}
        </Button>
      </div>

      {jsonMode && (
        <div className='space-y-2'>
          <JsonCodeEditor
            value={jsonValue}
            onChange={handleJsonChange}
            name='VideoTierPrice'
          />
          {jsonError ? (
            <p className='text-xs text-destructive'>{t(jsonError)}</p>
          ) : null}
        </div>
      )}

      {!jsonMode && entries.length === 0 && (
        <div className='rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground'>
          {t('No video model pricing configured yet.')}
        </div>
      )}

      {!jsonMode && entries.length > 0 && (
        <div className='space-y-4'>
          {entries.map((entry, index) => {
            const isCollapsed = collapsedIds.has(entry.id)
            return (
              <div key={entry.id} className='space-y-3 rounded-lg border p-4'>
                <div className='flex items-center gap-2'>
                  <Button
                    type='button'
                    variant='ghost'
                    size='sm'
                    className='h-8 w-8 p-0'
                    aria-expanded={!isCollapsed}
                    aria-label={
                      isCollapsed ? t('Expand model') : t('Collapse model')
                    }
                    onClick={() => toggleCollapsed(entry.id)}
                  >
                    {isCollapsed ? (
                      <ChevronRight className='h-4 w-4' aria-hidden='true' />
                    ) : (
                      <ChevronDown className='h-4 w-4' aria-hidden='true' />
                    )}
                  </Button>
                  <Input
                    list={MODEL_LIST_ID}
                    value={entry.model}
                    onChange={(event) =>
                      updateModelName(index, event.target.value)
                    }
                    placeholder={t('Model name')}
                    aria-label={t('Model name')}
                    className='max-w-md'
                  />
                  {isCollapsed && (
                    <span className='text-xs text-muted-foreground'>
                      {t('{{count}} tiers configured', {
                        count: countConfiguredTiers(entry),
                      })}
                    </span>
                  )}
                  <Button
                    type='button'
                    variant='ghost'
                    size='sm'
                    onClick={() => removeModel(index)}
                  >
                    <Trash2 data-icon='inline-start' />
                    {t('Remove model')}
                  </Button>
                </div>

                {!isCollapsed && (
                  <div className='overflow-x-auto'>
                    <table className='w-full max-w-2xl text-sm'>
                      <thead>
                        <tr className='text-left text-xs text-muted-foreground'>
                          <th className='w-20 py-1.5 pr-3 font-medium'>
                            {t('Resolution')}
                          </th>
                          <th className='py-1.5 pr-3 font-medium'>
                            {t('Without video input')}
                            <span className='block text-[11px] font-normal'>
                              {t('CNY per 1M tokens')}
                            </span>
                          </th>
                          <th className='py-1.5 font-medium'>
                            {t('With video input')}
                            <span className='block text-[11px] font-normal'>
                              {t('CNY per 1M tokens')}
                            </span>
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {VIDEO_TIER_RESOLUTIONS.map((resolution) => (
                          <tr key={resolution} className='border-t'>
                            <td className='w-20 py-1.5 pr-3 font-mono text-xs'>
                              {resolution}
                            </td>
                            {[false, true].map((hasVideo) => {
                              const key = videoTierKey(resolution, hasVideo)
                              return (
                                <td key={key} className='py-1.5 pr-3'>
                                  <Input
                                    type='number'
                                    min='0'
                                    step='any'
                                    inputMode='decimal'
                                    className='h-8 w-full'
                                    value={entry.prices[key] ?? ''}
                                    placeholder={t(
                                      'Not set (base tier, no multiplier)'
                                    )}
                                    aria-label={`${entry.model || t('Model name')} ${key}`}
                                    onChange={(event) =>
                                      updatePrice(
                                        index,
                                        key,
                                        event.target.value
                                      )
                                    }
                                  />
                                </td>
                              )
                            })}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      <datalist id={MODEL_LIST_ID}>
        {candidateModels.map((model) => (
          <option key={model} value={model} />
        ))}
      </datalist>
    </div>
  )
}
