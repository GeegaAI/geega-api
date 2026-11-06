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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { LegalVersionsPanel } from '../legal-versions-panel'

const apiGetMock = vi.fn()
const apiPostMock = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => apiGetMock(...args),
    post: (...args: unknown[]) => apiPostMock(...args),
  },
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
  fallbackLng: 'en',
})

function PanelHarness() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: false } },
      })
  )
  return (
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <LegalVersionsPanel />
      </I18nextProvider>
    </QueryClientProvider>
  )
}

beforeEach(() => {
  vi.useFakeTimers({ shouldAdvanceTime: true })
})

afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

describe('LegalVersionsPanel', () => {
  test('shows the current published version and the publish history', async () => {
    apiGetMock.mockResolvedValue({
      data: {
        success: true,
        data: {
          current_version: 3,
          versions: [
            {
              version: 3,
              published_at: 1761400000,
              published_by: 1,
              publisher_username: 'admin',
            },
          ],
        },
      },
    })

    render(<PanelHarness />)

    await waitFor(() => {
      expect(screen.getByTestId('legal-current-version')).toHaveTextContent(
        'v3'
      )
    })
    expect(screen.getByText('Version history')).toBeInTheDocument()
    expect(screen.getByText(/v3 ·/)).toBeInTheDocument()
  })

  test('publishing a new version posts to the publish endpoint and refreshes the panel', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    apiGetMock
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: {
            current_version: 1,
            versions: [],
          },
        },
      })
      .mockResolvedValue({
        data: {
          success: true,
          data: {
            current_version: 2,
            versions: [
              {
                version: 2,
                published_at: 1761400000,
                published_by: 1,
                publisher_username: 'admin',
              },
            ],
          },
        },
      })
    apiPostMock.mockResolvedValue({
      data: { success: true, data: { version: 2 } },
    })

    render(<PanelHarness />)

    const publishButtons = await screen.findAllByRole('button', {
      name: /Publish new version/i,
    })
    await user.click(publishButtons[0])

    const confirmDialog = await screen.findByRole('dialog')
    expect(confirmDialog).toBeInTheDocument()

    const confirmButtons = await screen.findAllByRole('button', {
      name: /Publish new version/i,
    })
    const confirmButton = confirmButtons.at(-1)
    if (!confirmButton) throw new Error('publish confirm button not found')
    await user.click(confirmButton)

    await waitFor(() => {
      expect(apiPostMock).toHaveBeenCalledWith('/api/legal/publish_version')
    })
    await waitFor(() => {
      expect(screen.getByTestId('legal-current-version')).toHaveTextContent(
        'v2'
      )
    })
  })
})
