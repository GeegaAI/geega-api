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
import { Columns2, Eye, FileText, Pencil } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type MarkdownEditorMode = 'edit' | 'split' | 'preview'

export type MarkdownEditorProps = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  disabled?: boolean
  heightClassName?: string
  ariaLabel?: string
  className?: string
}

const MODE_OPTIONS: Array<{
  mode: MarkdownEditorMode
  labelKey: string
  icon: typeof Eye
}> = [
  { mode: 'edit', labelKey: 'Edit', icon: Pencil },
  { mode: 'split', labelKey: 'Split', icon: Columns2 },
  { mode: 'preview', labelKey: 'Preview', icon: Eye },
]

export function MarkdownEditor(props: MarkdownEditorProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<MarkdownEditorMode>('split')

  return (
    <div
      className={cn(
        'border-input bg-background focus-within:border-ring focus-within:ring-ring/50 overflow-hidden rounded-lg border transition-colors focus-within:ring-3',
        props.className
      )}
    >
      <div className='bg-muted/30 flex h-8 items-center justify-between border-b px-2'>
        <div className='text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs font-medium'>
          <FileText className='h-3.5 w-3.5' aria-hidden='true' />
          <span>{t('Markdown')}</span>
        </div>
        <div
          className='flex items-center gap-0.5'
          role='group'
          aria-label={t('Markdown editor view mode')}
        >
          {MODE_OPTIONS.map(({ mode: optionMode, labelKey, icon: Icon }) => (
            <Button
              key={optionMode}
              type='button'
              variant='ghost'
              size='sm'
              className={cn(
                'h-6 gap-1 px-2 text-xs',
                mode === optionMode && 'bg-muted'
              )}
              aria-pressed={mode === optionMode}
              onClick={() => setMode(optionMode)}
            >
              <Icon className='h-3.5 w-3.5' aria-hidden='true' />
              {t(labelKey)}
            </Button>
          ))}
        </div>
      </div>
      <div className={cn('flex', props.heightClassName ?? 'h-72')}>
        <div
          className={cn(
            'min-w-0 overflow-hidden',
            mode === 'preview' ? 'hidden' : 'w-full',
            mode === 'split' && 'md:w-1/2 md:border-e'
          )}
        >
          <textarea
            value={props.value}
            onChange={(event) => props.onChange(event.target.value)}
            placeholder={props.placeholder}
            disabled={props.disabled}
            aria-label={props.ariaLabel}
            spellCheck={false}
            className={cn(
              'placeholder:text-muted-foreground h-full w-full resize-none overflow-y-auto bg-transparent px-3 py-2 font-mono text-xs leading-5 outline-none',
              'disabled:bg-input/30 disabled:cursor-not-allowed disabled:opacity-70'
            )}
          />
        </div>
        {mode !== 'edit' && (
          <div
            className={cn(
              'bg-background min-w-0 overflow-y-auto px-4 py-3',
              mode === 'preview' ? 'w-full' : 'hidden md:block md:w-1/2'
            )}
            aria-label={t('Markdown preview')}
          >
            {props.value.trim() ? (
              <RichContent
                mode='markdown'
                content={props.value}
                className='prose-neutral dark:prose-invert max-w-none text-sm'
              />
            ) : (
              <p className='text-muted-foreground text-sm'>
                {t('Nothing to preview')}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
