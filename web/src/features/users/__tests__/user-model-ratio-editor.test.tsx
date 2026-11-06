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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { getPricing } from '@/features/pricing/api'
import type { PricingData, PricingModel } from '@/features/pricing/types'

import {
  getUserModelRatios,
  setUserModelRatios,
} from '../api'
import { UserModelRatioEditor } from '../components/user-model-ratio-editor'
import { toast } from 'sonner'

vi.mock('../api', () => ({
  getUserModelRatios: vi.fn(),
  setUserModelRatios: vi.fn(),
}))
vi.mock('@/features/pricing/api', () => ({
  getPricing: vi.fn(),
}))
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')

const mockGetUserModelRatios = vi.mocked(getUserModelRatios)
const mockSetUserModelRatios = vi.mocked(setUserModelRatios)
const mockGetPricing = vi.mocked(getPricing)

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Custom ratio': 'Custom discount',
        'Custom ratio: {{model}}': 'Custom discount: {{model}}',
        'Failed to load model ratios': 'Failed to load model discounts',
        'Failed to load pricing data': 'Failed to load pricing data',
        'Failed to save model ratio overrides':
          'Failed to save model discounts',
        'Global value': 'Global value',
        'Inherit global price': 'No discount',
        'Inherit global ratio': 'No discount',
        'Leave empty to inherit the global price. Set 0 to make this model free for this user.':
          'Leave empty for no discount (billed at the global price). Set 0 to make this model free for this user.',
        'For tiered and per-request models, the custom ratio acts as a coefficient applied on top of the global price.':
          'For tiered and per-request models, the custom discount acts as a coefficient applied on top of the global price.',
        'Loading model ratios...': 'Loading model discounts...',
        Model: 'Model',
        'Model ratio overrides saved': 'Model discounts saved',
        'No matching models.': 'No matching models.',
        'No models available for this group.':
          'No models available for this group.',
        'Only show overridden': 'Only show overridden',
        Overridden: 'Overridden',
        'Page {{page}} of {{total}}': 'Page {{page}} of {{total}}',
        'Per-request': 'Per-request',
        request: 'request',
        'Previous page': 'Previous page',
        'Discount must be a number greater than or equal to 0 and less than 1':
          'Discount must be a number greater than or equal to 0 and less than 1',
        Retry: 'Retry',
        Save: 'Save',
        'Saving...': 'Saving...',
        'Search models...': 'Search models...',
        Tiered: 'Tiered',
      },
    },
  },
})

function makeModel(
  overrides: Partial<PricingModel> & { model_name: string }
): PricingModel {
  return {
    id: 1,
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    ...overrides,
  }
}

const pricingFixture: PricingData = {
  success: true,
  data: [
    makeModel({ model_name: 'gpt-a', model_ratio: 2 }),
    makeModel({ model_name: 'gpt-b', model_ratio: 3 }),
    makeModel({
      model_name: 'per-call',
      quota_type: 1,
      model_ratio: 1,
      model_price: 0.1,
    }),
    makeModel({
      model_name: 'tiered-model',
      model_ratio: 1,
      billing_mode: 'tiered_expr',
    }),
    makeModel({ model_name: 'vip-only', enable_groups: ['vip'] }),
  ],
  vendors: [],
  group_ratio: {},
  usable_group: {},
  supported_endpoint: {},
  auto_groups: [],
}

async function renderEditor(propsOverrides?: {
  group?: string
  userId?: number
  waitForLabel?: string
}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <UserModelRatioEditor
          userId={propsOverrides?.userId ?? 7}
          group={propsOverrides?.group ?? 'default'}
        />
      </I18nextProvider>
    </QueryClientProvider>
  )
  // Wait until data loaded and rows are rendered
  await screen.findByLabelText(propsOverrides?.waitForLabel ?? 'Custom discount: gpt-a')
}

beforeEach(() => {
  mockGetUserModelRatios.mockResolvedValue({
    success: true,
    data: { 'gpt-a': 0.5 },
  })
  mockGetPricing.mockResolvedValue(pricingFixture)
  mockSetUserModelRatios.mockResolvedValue({ success: true, data: {} })
})

describe('UserModelRatioEditor', () => {
  test('clearing an override and saving removes the discount (empty ratios payload)', async () => {
    await renderEditor()

    const inputA = screen.getByLabelText('Custom discount: gpt-a')
    expect(inputA).toHaveValue(0.5)
    fireEvent.change(inputA, { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(mockSetUserModelRatios).toHaveBeenCalledWith(7, {})
    })
  })

  test('per-request models accept a price coefficient and show the global price', async () => {
    await renderEditor()

    const perCallInput = screen.getByLabelText('Custom discount: per-call')
    expect(perCallInput).toBeEnabled()
    expect(perCallInput).toHaveAttribute('placeholder', 'No discount')
    expect(screen.getByText('$0.1 / request')).toBeInTheDocument()
  })

  test('tiered billing models accept a settlement coefficient discount', async () => {
    await renderEditor()

    const tieredInput = screen.getByLabelText('Custom discount: tiered-model')
    expect(tieredInput).toBeEnabled()
    expect(tieredInput).toHaveAttribute('placeholder', 'No discount')
    expect(screen.getByText('Tiered')).toBeInTheDocument()
  })

  test('an invalid discount blocks saving and shows a validation message', async () => {
    await renderEditor()

    const inputB = screen.getByLabelText('Custom discount: gpt-b')
    for (const invalid of ['-1', '1', '1.5']) {
      fireEvent.change(inputB, { target: { value: invalid } })
      expect(
        screen.getByText(
          'Discount must be a number greater than or equal to 0 and less than 1'
        )
      ).toBeInTheDocument()
    }

    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        'Discount must be a number greater than or equal to 0 and less than 1'
      )
    })
    expect(mockSetUserModelRatios).not.toHaveBeenCalled()
  })

  test('saving sends a full replacement payload merging kept and new overrides', async () => {
    await renderEditor()

    const inputB = screen.getByLabelText('Custom discount: gpt-b')
    fireEvent.change(inputB, { target: { value: '0.25' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(mockSetUserModelRatios).toHaveBeenCalledWith(7, {
        'gpt-a': 0.5,
        'gpt-b': 0.25,
      })
    })
    expect(toast.success).toHaveBeenCalledWith('Model discounts saved')
  })

  test('lists only models enabled for the target user group', async () => {
    await renderEditor({ group: 'vip', waitForLabel: 'Custom discount: vip-only' })

    await screen.findByLabelText('Custom discount: vip-only')
    expect(screen.queryByLabelText('Custom discount: gpt-a')).toBeNull()
    expect(screen.queryByLabelText('Custom discount: per-call')).toBeNull()
  })

  test('search filters rows by model name', async () => {
    await renderEditor()

    fireEvent.change(screen.getByPlaceholderText('Search models...'), {
      target: { value: 'gpt-b' },
    })

    await waitFor(() => {
      expect(screen.queryByLabelText('Custom discount: gpt-a')).toBeNull()
    })
    expect(screen.getByLabelText('Custom discount: gpt-b')).toBeInTheDocument()
  })
})
