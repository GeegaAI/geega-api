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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { MarkdownEditor } from '@/components/markdown-editor'
import { Spinner } from '@/components/ui/spinner'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  LEGAL_DEFAULT_LANGUAGE,
  parseLegalLanguageValue,
  serializeLegalLanguageValue,
  updateLegalLanguageValue,
} from '@/features/legal/legal-language'
import { getLegalDocs, saveLegalDocs } from '@/features/legal/api'
import { INTERFACE_LANGUAGE_OPTIONS } from '@/i18n/languages'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { invalidateLegalStatusCache } from '../hooks/use-update-option'
import { useSettingsForm } from '../hooks/use-settings-form'
import { LegalVersionsPanel } from './legal-versions-panel'

type LegalEditorLanguage = {
  value: string
  lang: string
  label?: string
  labelKey?: string
}

const LEGAL_EDITOR_LANGUAGES: LegalEditorLanguage[] = [
  {
    value: 'default',
    lang: LEGAL_DEFAULT_LANGUAGE,
    labelKey: 'Default (fallback)',
  },
  ...INTERFACE_LANGUAGE_OPTIONS.map((language) => ({
    value: language.code,
    lang: language.code,
    label: language.label,
  })),
]

const LEGAL_DOC_FIELDS = [
  {
    contentField: 'user_agreement',
    nameField: 'user_agreement_name',
    titleKey: 'User Agreement',
    placeholderKey:
      'Provide Markdown, HTML, or an external URL for the user agreement',
    emptyHintKey:
      'Clear every language to disable the user agreement. Supports Markdown, HTML, or a full URL to redirect users.',
  },
  {
    contentField: 'privacy_policy',
    nameField: 'privacy_policy_name',
    titleKey: 'Privacy Policy',
    placeholderKey:
      'Provide Markdown, HTML, or an external URL for the privacy policy',
    emptyHintKey:
      'Clear every language to disable the privacy policy. Supports Markdown, HTML, or a full URL to redirect users.',
  },
  {
    contentField: 'service_agreement',
    nameField: 'service_agreement_name',
    titleKey: 'Product Service Agreement',
    placeholderKey:
      'Provide Markdown, HTML, or an external URL for the service agreement',
    emptyHintKey:
      'Clear every language to disable the service agreement. Supports Markdown, HTML, or a full URL to redirect users.',
  },
] as const

type LegalDocField = (typeof LEGAL_DOC_FIELDS)[number]

const legalSectionSchema = z.object({
  user_agreement: z.string(),
  privacy_policy: z.string(),
  service_agreement: z.string(),
  user_agreement_name: z.string(),
  privacy_policy_name: z.string(),
  service_agreement_name: z.string(),
})

type LegalSectionFormValues = z.infer<typeof legalSectionSchema>

function normalizeValue(value: unknown): string {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : String(value)
}

function buildFormDefaults(row: {
  user_agreement: string
  privacy_policy: string
  service_agreement: string
  user_agreement_name: string
  privacy_policy_name: string
  service_agreement_name: string
}): LegalSectionFormValues {
  return {
    user_agreement: serializeLegalLanguageValue(
      parseLegalLanguageValue(row.user_agreement)
    ),
    privacy_policy: serializeLegalLanguageValue(
      parseLegalLanguageValue(row.privacy_policy)
    ),
    service_agreement: serializeLegalLanguageValue(
      parseLegalLanguageValue(row.service_agreement)
    ),
    user_agreement_name: serializeLegalLanguageValue(
      parseLegalLanguageValue(row.user_agreement_name)
    ),
    privacy_policy_name: serializeLegalLanguageValue(
      parseLegalLanguageValue(row.privacy_policy_name)
    ),
    service_agreement_name: serializeLegalLanguageValue(
      parseLegalLanguageValue(row.service_agreement_name)
    ),
  }
}

export function LegalSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [activeLanguageValue, setActiveLanguageValue] = useState('default')
  const [activeDocField, setActiveDocField] = useState<string>(
    LEGAL_DOC_FIELDS[0].contentField
  )
  const activeLanguage =
    LEGAL_EDITOR_LANGUAGES.find(
      (language) => language.value === activeLanguageValue
    ) ?? LEGAL_EDITOR_LANGUAGES[0]

  const { data, isLoading } = useQuery({
    queryKey: ['legal-docs'],
    queryFn: getLegalDocs,
  })
  const savedRow = data?.data

  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<LegalSectionFormValues>({
      resolver: zodResolver(legalSectionSchema) as Resolver<
        LegalSectionFormValues,
        unknown,
        LegalSectionFormValues
      >,
      // savedRow 到达前不挂表单，避免用空默认值初始化
      defaultValues: buildFormDefaults(savedRow ?? {
        user_agreement: '',
        privacy_policy: '',
        service_agreement: '',
        user_agreement_name: '',
        privacy_policy_name: '',
        service_agreement_name: '',
      }),
      onSubmit: async (values) => {
        const res = await saveLegalDocs({
          user_agreement: normalizeValue(values.user_agreement),
          privacy_policy: normalizeValue(values.privacy_policy),
          service_agreement: normalizeValue(values.service_agreement),
          user_agreement_name: normalizeValue(values.user_agreement_name),
          privacy_policy_name: normalizeValue(values.privacy_policy_name),
          service_agreement_name: normalizeValue(values.service_agreement_name),
        })
        toast.success(
          t('Saved as version {{version}}', {
            version: res?.data?.version ?? '',
          })
        )
        await queryClient.invalidateQueries({ queryKey: ['legal-docs'] })
        await invalidateLegalStatusCache(queryClient)
        form.reset(buildFormDefaults(values))
      },
    })

  if (isLoading || !savedRow) {
    return (
      <div className='flex items-center justify-center py-16'>
        <Spinner className='h-6 w-6' />
      </div>
    )
  }

  const languageItems = LEGAL_EDITOR_LANGUAGES.map((language) => ({
    value: language.value,
    label: language.label ?? t(language.labelKey ?? ''),
  }))

  return (
    <>
      <FormNavigationGuard when={isDirty} />

      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            onReset={handleReset}
            isSaving={isSubmitting}
            isResetDisabled={!isDirty}
          />
          <FormDirtyIndicator isDirty={isDirty} />

          <SettingsSection title={t('Legal Documents')}>
            <LegalVersionsPanel />
            <div className='space-y-2'>
              <div className='flex flex-wrap items-center gap-3'>
                <span className='text-sm font-medium'>
                  {t('Document language')}
                </span>
                <Select
                  items={languageItems}
                  value={activeLanguageValue}
                  onValueChange={(value) =>
                    setActiveLanguageValue(String(value))
                  }
                >
                  <SelectTrigger className='w-52'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {LEGAL_EDITOR_LANGUAGES.map((language) => (
                        <SelectItem key={language.value} value={language.value}>
                          {language.label ?? t(language.labelKey ?? '')}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Documents are stored per language. Users see the version matching their interface language; missing languages fall back to the default version, then English.'
                )}
              </p>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Users must read and agree to all enabled documents before signing in. A document with no content in any language is disabled.'
                )}
              </p>
            </div>
            <Tabs
              value={activeDocField}
              onValueChange={(value) => setActiveDocField(String(value))}
              className='mt-4 min-w-0 gap-3'
            >
              <TabsList className='grid w-full grid-cols-3'>
                {LEGAL_DOC_FIELDS.map((doc) => {
                  const docIsDirty = Boolean(
                    form.formState.dirtyFields[doc.contentField] ||
                    form.formState.dirtyFields[doc.nameField]
                  )
                  return (
                    <TabsTrigger
                      key={doc.contentField}
                      value={doc.contentField}
                      className='gap-1.5'
                    >
                      {t(doc.titleKey)}
                      {docIsDirty && (
                        <span
                          className='bg-primary size-1.5 rounded-full'
                          aria-hidden='true'
                        />
                      )}
                    </TabsTrigger>
                  )
                })}
              </TabsList>

              {LEGAL_DOC_FIELDS.map((doc) => (
                <TabsContent
                  key={doc.contentField}
                  value={doc.contentField}
                  keepMounted
                >
                  <SettingsFormGrid>
                    <LegalDocFields
                      doc={doc}
                      form={form}
                      activeLanguage={activeLanguage}
                      savedRaw={savedRow[doc.contentField]}
                    />
                  </SettingsFormGrid>
                </TabsContent>
              ))}
            </Tabs>
          </SettingsSection>
        </SettingsForm>
      </Form>
    </>
  )
}

type LegalDocFieldsProps = {
  doc: LegalDocField
  form: ReturnType<typeof useSettingsForm<LegalSectionFormValues>>['form']
  activeLanguage: LegalEditorLanguage
  savedRaw: string
}

function LegalDocFields(props: LegalDocFieldsProps) {
  const { t } = useTranslation()
  const { doc, activeLanguage } = props
  const savedValues = parseLegalLanguageValue(props.savedRaw)

  const savedLabels: string[] = []
  if ((savedValues[LEGAL_DEFAULT_LANGUAGE] ?? '').trim()) {
    savedLabels.push(t('Default (fallback)'))
  }
  for (const language of INTERFACE_LANGUAGE_OPTIONS) {
    if ((savedValues[language.code] ?? '').trim()) {
      savedLabels.push(language.label)
    }
  }
  const savedLine =
    savedLabels.length > 0
      ? t('Saved languages: {{languages}}', {
          languages: savedLabels.join(' / '),
        })
      : t('No saved content for this document yet.')

  const languageLabel = activeLanguage.label ?? t(activeLanguage.labelKey ?? '')

  return (
    <SettingsFormGridItem span='full'>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-baseline justify-between gap-2'>
          <h4 className='text-sm font-medium'>{t(doc.titleKey)}</h4>
          <p className='text-muted-foreground text-xs'>{savedLine}</p>
        </div>
        <FormField
          control={props.form.control}
          name={doc.nameField}
          render={({ field }) => {
            const names = parseLegalLanguageValue(field.value)
            return (
              <FormItem>
                <FormLabel>{t('Display name')}</FormLabel>
                <FormControl>
                  <Input
                    value={names[activeLanguage.lang] ?? ''}
                    onChange={(event) =>
                      field.onChange(
                        updateLegalLanguageValue(
                          field.value,
                          activeLanguage.lang,
                          event.target.value
                        )
                      )
                    }
                    placeholder={t(doc.titleKey)}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Document name shown to users for this language. Leave empty to use the default name.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )
          }}
        />
        <FormField
          control={props.form.control}
          name={doc.contentField}
          render={({ field }) => {
            const contents = parseLegalLanguageValue(field.value)
            const value = contents[activeLanguage.lang] ?? ''
            return (
              <FormItem>
                <FormLabel>{t('Content')}</FormLabel>
                {value.trim() === '' && savedLabels.length > 0 && (
                  <p className='text-muted-foreground text-xs'>
                    {t(
                      'This language is empty. Users will fall back to the default content, then English.'
                    )}
                  </p>
                )}
                <FormControl>
                  <MarkdownEditor
                    value={value}
                    onChange={(nextValue) =>
                      field.onChange(
                        updateLegalLanguageValue(
                          field.value,
                          activeLanguage.lang,
                          nextValue
                        )
                      )
                    }
                    placeholder={t(doc.placeholderKey)}
                    ariaLabel={`${t(doc.titleKey)} - ${languageLabel}`}
                    heightClassName='h-144'
                  />
                </FormControl>
                <FormDescription>{t(doc.emptyHintKey)}</FormDescription>
                <FormMessage />
              </FormItem>
            )
          }}
        />
      </div>
    </SettingsFormGridItem>
  )
}

