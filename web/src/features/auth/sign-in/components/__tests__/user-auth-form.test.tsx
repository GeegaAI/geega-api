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
import { beforeEach, describe, expect, test, vi } from 'vitest'

import type { SystemStatus } from '../../../types'

const loginMock = vi.fn()

vi.mock('@/features/auth/api', () => ({
  login: (...args: unknown[]) => loginMock(...args),
  wechatLoginByCode: vi.fn(),
}))

const statusMocks: Array<{ status: SystemStatus | null }> = []

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => statusMocks.at(-1) ?? { status: null },
}))

const handleLoginSuccessMock = vi.fn()

vi.mock('@/features/auth/hooks/use-auth-redirect', () => ({
  useAuthRedirect: () => ({
    handleLoginSuccess: handleLoginSuccessMock,
    redirectTo2FA: vi.fn(),
    redirectToLogin: vi.fn(),
    redirectToRegister: vi.fn(),
  }),
}))

vi.mock('@/features/auth/hooks/use-turnstile', () => ({
  useTurnstile: () => ({
    isTurnstileEnabled: false,
    turnstileSiteKey: '',
    turnstileToken: '',
    setTurnstileToken: vi.fn(),
    validateTurnstile: () => true,
  }),
}))

vi.mock('@/features/auth/passkey', () => ({
  beginPasskeyLogin: vi.fn(),
  finishPasskeyLogin: vi.fn(),
}))

vi.mock('@/lib/passkey', () => ({
  buildAssertionResult: vi.fn(),
  prepareCredentialRequestOptions: vi.fn(),
  isPasskeySupported: vi.fn().mockResolvedValue(false),
}))

const isAuthBundleMock = vi.fn()

vi.mock('@/lib/api', () => ({
  api: { get: vi.fn(), post: vi.fn() },
  isAuthBundle: (...args: unknown[]) => isAuthBundleMock(...args),
}))

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { to: string; children?: React.ReactNode }) => (
    <a href={props.to}>{props.children}</a>
  ),
  useNavigate: () => vi.fn(),
}))

const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { Toaster } = await import('sonner')
const { UserAuthForm } = await import('../user-auth-form')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
  fallbackLng: 'en',
})

function makeStatus(overrides: Record<string, unknown>): SystemStatus {
  return overrides as SystemStatus
}

function setUseStatus(status: SystemStatus | null) {
  statusMocks.push({ status })
}

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <UserAuthForm />
        <Toaster position='top-center' />
      </I18nextProvider>
    </QueryClientProvider>
  )
}

const DOCS_STATUS = makeStatus({
  password_login_enabled: true,
  user_agreement_enabled: true,
  privacy_policy_enabled: true,
  service_agreement_enabled: true,
})

beforeEach(() => {
  statusMocks.length = 0
  loginMock.mockReset()
  handleLoginSuccessMock.mockReset()
  isAuthBundleMock.mockReset()
})

describe('UserAuthForm legal consent gating', () => {
  test('keeps the consent checkbox unchecked by default until the user checks it', async () => {
    const user = userEvent.setup()
    setUseStatus(DOCS_STATUS)

    renderForm()

    const signInButton = screen.getByRole('button', { name: 'Sign in' })
    expect(screen.getByRole('checkbox')).not.toBeChecked()
    expect(signInButton).toBeDisabled()

    await user.click(screen.getByRole('checkbox'))

    expect(signInButton).toBeEnabled()
  })

  test('sends the plain login request once the consent checkbox is checked', async () => {
    const user = userEvent.setup()
    setUseStatus(DOCS_STATUS)
    loginMock.mockResolvedValue({
      success: true,
      message: '',
      data: { access_token: 'token', user: { id: 1 } },
    })
    isAuthBundleMock.mockReturnValue(true)

    renderForm()

    await user.type(screen.getByLabelText('Username or Email'), 'alice')
    await user.type(screen.getByLabelText('Password'), 'password')
    await user.click(screen.getByRole('checkbox'))
    await user.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => {
      expect(loginMock).toHaveBeenCalledWith(
        expect.objectContaining({ username: 'alice' })
      )
    })
    expect(handleLoginSuccessMock).toHaveBeenCalled()
  })

  test('does not gate the sign-in button when no legal document is enabled', () => {
    setUseStatus(makeStatus({ password_login_enabled: true }))

    renderForm()

    expect(screen.getByRole('button', { name: 'Sign in' })).toBeEnabled()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  })
})
