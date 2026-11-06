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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { beforeAll, describe, expect, test } from 'vitest'

import { VideoTierPriceForm } from '../video-tier-price-form'

function renderForm(defaultValue: string) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <VideoTierPriceForm defaultValue={defaultValue} />
    </QueryClientProvider>
  )
}

describe('video tier price collapse', () => {
  beforeAll(async () => {
    await i18next.init({
      lng: 'en',
      fallbackLng: 'en',
      interpolation: { escapeValue: false },
    })
    i18next.addResourceBundle('en', 'translation', {
      'Add model': 'Add model',
      'Collapse model': 'Collapse model',
      'CNY per 1M tokens': 'CNY per 1M tokens',
      'Not set (base tier, no multiplier)':
        'Not set (base tier, no multiplier)',
      'Expand model': 'Expand model',
      'Model name': 'Model name',
      'No video model pricing configured yet.':
        'No video model pricing configured yet.',
      'Remove model': 'Remove model',
      Resolution: 'Resolution',
      'Save video model pricing': 'Save video model pricing',
      'Saving...': 'Saving...',
      'Switch to JSON': 'Switch to JSON',
      'Switch to Visual': 'Switch to Visual',
      'With video input': 'With video input',
      'Without video input': 'Without video input',
      '{{count}} tiers configured': '{{count}} tiers configured',
    })
  })

  test('collapses configured models and expands them on demand', () => {
    renderForm('{"model-a":{"720p":46}}')

    expect(screen.getByText('1 tiers configured')).toBeInTheDocument()
    expect(screen.queryByRole('spinbutton', { name: 'model-a 720p' })).toBeNull()

    const toggle = screen.getByRole('button', { name: 'Expand model' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(toggle)

    expect(
      screen.getByRole('button', { name: 'Collapse model' })
    ).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('spinbutton', { name: 'model-a 720p' })).toHaveValue(
      46
    )
  })

  test('labels both price columns with the CNY per 1M tokens unit', () => {
    renderForm('{"model-a":{"720p":46}}')

    fireEvent.click(screen.getByRole('button', { name: 'Expand model' }))

    expect(screen.getAllByText('CNY per 1M tokens')).toHaveLength(2)
  })

  test('shows the base-tier fallback hint on unset price cells', () => {
    renderForm('{"model-a":{"720p":46}}')

    fireEvent.click(screen.getByRole('button', { name: 'Expand model' }))

    expect(
      screen.getAllByPlaceholderText('Not set (base tier, no multiplier)')
    ).toHaveLength(11)
  })

  test('keeps a newly added model expanded', () => {
    renderForm('{}')

    fireEvent.click(screen.getByRole('button', { name: 'Add model' }))

    expect(
      screen.getByRole('button', { name: 'Collapse model' })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('spinbutton', { name: 'Model name 720p' })
    ).toBeInTheDocument()
  })
})
