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
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

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

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsControlGroup,
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const quotaResetSchema = z.object({
  general_setting: z.object({
    quota_reset_day: z.coerce.number().min(1).max(28),
    quota_reset_hour: z.coerce.number().min(0).max(23),
    quota_reset_cycle_days: z.coerce.number().min(1).max(365),
  }),
})

type QuotaResetFormValues = z.infer<typeof quotaResetSchema>
type QuotaResetInputValue = number | ''

function diffChanged(
  before: QuotaResetFormValues,
  after: QuotaResetFormValues
): Record<string, string | number> {
  const out: Record<string, string | number> = {}
  const keys = [
    'quota_reset_day',
    'quota_reset_hour',
    'quota_reset_cycle_days',
  ] as const
  for (const key of keys) {
    if (
      String(before.general_setting[key]) !== String(after.general_setting[key])
    ) {
      out[`general_setting.${key}`] = after.general_setting[key]
    }
  }
  return out
}

export function QuotaResetSettingsSection(props: {
  defaultValues: QuotaResetFormValues
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<QuotaResetFormValues>({
      resolver: zodResolver(quotaResetSchema) as Resolver<
        QuotaResetFormValues,
        unknown,
        QuotaResetFormValues
      >,
      defaultValues: props.defaultValues,
      onSubmit: async (data) => {
        for (const [key, value] of Object.entries(
          diffChanged(props.defaultValues, data)
        )) {
          await updateOption.mutateAsync({
            key,
            value: value as string | number | boolean,
          })
        }
      },
    })

  const numberInputHandler =
    (onChange: (value: QuotaResetInputValue) => void) =>
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber
      onChange(Number.isNaN(value) ? '' : value)
    }

  return (
    <SettingsSection title={t('Quota Reset Policy')}>
      <FormNavigationGuard when={isDirty} />
      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending || isSubmitting}
          />
          <FormDirtyIndicator isDirty={isDirty} />
          <SettingsFormGrid>
            <SettingsFormGridItem span='full'>
              <SettingsControlGroup className='space-y-4'>
                <h4 className='text-sm font-medium'>
                  {t('Monthly fixed-day reset time configuration')}
                </h4>
                <div className='grid gap-5 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='general_setting.quota_reset_day'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Monthly reset day')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            min={1}
                            max={28}
                            value={field.value ?? ''}
                            onChange={numberInputHandler(field.onChange)}
                            name={field.name}
                            onBlur={field.onBlur}
                            ref={field.ref}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Date of the monthly fixed-day reset (day 1-28, server time). Applies only to users on the monthly fixed-day reset policy.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name='general_setting.quota_reset_hour'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Monthly reset hour')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            min={0}
                            max={23}
                            value={field.value ?? ''}
                            onChange={numberInputHandler(field.onChange)}
                            name={field.name}
                            onBlur={field.onBlur}
                            ref={field.ref}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            'Hour of the monthly fixed-day reset (0-23, server time), combined with the reset day to determine when the reset runs.'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </SettingsControlGroup>
            </SettingsFormGridItem>

            <SettingsFormGridItem span='full'>
              <SettingsControlGroup className='space-y-4'>
                <h4 className='text-sm font-medium'>
                  {t('Billing-cycle reset time configuration')}
                </h4>
                <div className='grid gap-5 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name='general_setting.quota_reset_cycle_days'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Cycle length (days)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            min={1}
                            max={365}
                            value={field.value ?? ''}
                            onChange={numberInputHandler(field.onChange)}
                            name={field.name}
                            onBlur={field.onBlur}
                            ref={field.ref}
                          />
                        </FormControl>
                        <FormDescription>
                          {t(
                            "Cycle length in days (1-365), counted from the user's quota effective time. Applies only to users on the billing-cycle reset policy; remaining quota and top-ups made during the cycle are cleared when the cycle ends."
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </SettingsControlGroup>
            </SettingsFormGridItem>

            <SettingsFormGridItem span='full'>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Business note: guide cycle/no-reset customers to use operation grants instead of self top-up — a cycle reset would also clear topped-up quota.'
                )}
              </p>
            </SettingsFormGridItem>
          </SettingsFormGrid>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
