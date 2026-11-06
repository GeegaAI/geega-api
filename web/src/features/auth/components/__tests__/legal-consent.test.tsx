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
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import type { SystemStatus } from '../../types'

const apiGetMock = vi.fn()

vi.mock('@/lib/api', () => ({
  api: {
    get: (...args: unknown[]) => apiGetMock(...args),
  },
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { LegalConsent } = await import('../legal-consent')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
  fallbackLng: 'en',
})

function makeStatus(overrides: Record<string, unknown>): SystemStatus {
  return overrides as SystemStatus
}

const ALL_DOCS_STATUS = makeStatus({
  user_agreement_enabled: true,
  user_agreement_name: 'Custom User Agreement Name',
  privacy_policy_enabled: true,
  service_agreement_enabled: true,
  service_agreement_name: 'Custom Service Agreement Name',
})

function ConsentHarness(props: {
  status: SystemStatus
  agreed?: Record<string, boolean>
  onChange: (key: string, value: boolean) => void
}) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: false } },
      })
  )
  const [agreed, setAgreed] = useState<Record<string, boolean>>(
    props.agreed ?? {}
  )
  const handleChange = (key: string, value: boolean) => {
    setAgreed((prev) => ({ ...prev, [key]: value }))
    props.onChange(key, value)
  }
  return (
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <LegalConsent
          status={props.status}
          agreed={agreed}
          onAgreedChange={handleChange}
        />
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

describe('LegalConsent', () => {
  test('renders nothing when no legal document is enabled', () => {
    const onChange = vi.fn()
    const { container } = render(
      <ConsentHarness status={makeStatus({})} onChange={onChange} />
    )
    expect(container).toBeEmptyDOMElement()
  })

  test('renders one checkbox listing every enabled document by display name', () => {
    render(<ConsentHarness status={ALL_DOCS_STATUS} onChange={vi.fn()} />)

    expect(screen.getAllByRole('checkbox')).toHaveLength(1)
    expect(screen.getByText('Custom User Agreement Name')).toBeInTheDocument()
    expect(screen.getByText('Privacy Policy')).toBeInTheDocument()
    expect(
      screen.getByText('Custom Service Agreement Name')
    ).toBeInTheDocument()
  })

  test('checking the checkbox marks every enabled document as agreed at once', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    const agreedChange = vi.fn()
    render(<ConsentHarness status={ALL_DOCS_STATUS} onChange={agreedChange} />)

    await user.click(screen.getByRole('checkbox'))

    expect(agreedChange).toHaveBeenCalledWith('user_agreement', true)
    expect(agreedChange).toHaveBeenCalledWith('privacy_policy', true)
    expect(agreedChange).toHaveBeenCalledWith('service_agreement', true)

    await user.click(screen.getByRole('checkbox'))

    expect(agreedChange).toHaveBeenCalledWith('user_agreement', false)
    expect(agreedChange).toHaveBeenCalledWith('privacy_policy', false)
    expect(agreedChange).toHaveBeenCalledWith('service_agreement', false)
  })

  test('opens the reading dialog with the document content when a document name is clicked', async () => {
    apiGetMock.mockResolvedValue({
      data: { success: true, message: '', data: '# Agreement\n\nBody text.' },
    })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    render(<ConsentHarness status={ALL_DOCS_STATUS} onChange={vi.fn()} />)

    await user.click(screen.getByText('Custom User Agreement Name'))

    expect(apiGetMock).toHaveBeenCalledWith('/api/user-agreement?lang=en')
    await screen.findByText('Body text.')
  })
})
