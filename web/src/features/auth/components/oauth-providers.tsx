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
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import {
  IconDiscord,
  IconLinuxDo,
  IconTelegram,
  IconWeChat,
} from '@/assets/brand-icons'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { useOAuthLogin } from '../hooks/use-oauth-login'
import type { SystemStatus } from '../types'
import { TelegramLoginDialog } from './telegram-login-dialog'

type OAuthProvidersProps = {
  status: SystemStatus | null
  disabled?: boolean
  consentLocked?: boolean
  className?: string
  onWeChatLogin?: () => void
  isWeChatLoading?: boolean
  redirectTo?: string
}

type ProviderButton = {
  key: string
  label: string
  onClick: () => void
  icon?: ReactNode
  disabled?: boolean
}

export function OAuthProviders({
  status,
  disabled = false,
  consentLocked = false,
  className,
  onWeChatLogin,
  isWeChatLoading = false,
  redirectTo,
}: OAuthProvidersProps) {
  const { t } = useTranslation()
  // 勾选被锁定时点击按钮，用同一个 Tooltip 提示（与悬停展示一致），不弹单独 message
  const [hintOpenKey, setHintOpenKey] = useState<string | null>(null)
  const {
    isLoading,
    handleDiscordLogin,
    handleOIDCLogin,
    handleLinuxDOLogin,
    handleTelegramLogin,
    handleCustomOAuthLogin,
    isTelegramDialogOpen,
    isTelegramPending,
    handleTelegramAuthorization,
    setIsTelegramDialogOpen,
  } = useOAuthLogin(status, redirectTo)

  const providerButtons: ProviderButton[] = []

  if (status?.wechat_login && onWeChatLogin) {
    providerButtons.push({
      key: 'wechat',
      label: t('Continue with WeChat'),
      onClick: onWeChatLogin,
      icon: <IconWeChat className='h-4 w-4' />,
      disabled: isWeChatLoading,
    })
  }

  if (status?.discord_oauth) {
    providerButtons.push({
      key: 'discord',
      label: t('Continue with Discord'),
      onClick: handleDiscordLogin,
      icon: <IconDiscord className='h-4 w-4' />,
    })
  }

  if (status?.oidc_enabled) {
    const oidcDisplayName = status.oidc_display_name?.trim() || 'OIDC'
    providerButtons.push({
      key: 'oidc',
      label: t('Continue with {{name}}', {
        name: oidcDisplayName,
      }),
      onClick: handleOIDCLogin,
    })
  }

  if (status?.linuxdo_oauth) {
    providerButtons.push({
      key: 'linuxdo',
      label: t('Continue with LinuxDO'),
      onClick: handleLinuxDOLogin,
      icon: <IconLinuxDo className='h-4 w-4' />,
    })
  }

  if (status?.telegram_oauth) {
    providerButtons.push({
      key: 'telegram',
      label: t('Continue with Telegram'),
      onClick: handleTelegramLogin,
      icon: <IconTelegram data-icon='inline-start' />,
    })
  }

  // Custom OAuth providers
  const customProviders = status?.custom_oauth_providers
  if (customProviders && customProviders.length > 0) {
    for (const provider of customProviders) {
      providerButtons.push({
        key: `custom-${provider.slug}`,
        label: t('Continue with {{name}}', { name: provider.name }),
        onClick: () => handleCustomOAuthLogin(provider),
      })
    }
  }

  if (providerButtons.length === 0) return null

  return (
    <>
      <div className={cn('space-y-3', className)}>
        {/* 无 Provider 时 Base UI 默认 hover 延迟约 600ms，短暂掠过按钮
            提示会被取消（表现为有时一直不展示）；delay=0 悬停立即触发 */}
        <TooltipProvider delay={0}>
          <div className='flex flex-col gap-2'>
            {providerButtons.map(
              ({ key, label, onClick, icon, disabled: extraDisabled }) => {
                const button = (
                  <Button
                    variant='outline'
                    type='button'
                    disabled={disabled || isLoading || extraDisabled}
                    onClick={() => {
                      if (consentLocked) {
                        setHintOpenKey(key)
                        return
                      }
                      onClick()
                    }}
                    className={cn(
                      'h-11 w-full justify-center gap-2 rounded-lg',
                      consentLocked && 'cursor-not-allowed opacity-60',
                      !consentLocked &&
                        'disabled:pointer-events-auto disabled:cursor-not-allowed'
                    )}
                  >
                    {icon}
                    {label}
                  </Button>
                )
                if (!consentLocked) {
                  return <div key={key}>{button}</div>
                }
                return (
                  <Tooltip
                    key={key}
                    open={hintOpenKey === key}
                    onOpenChange={(open) => setHintOpenKey(open ? key : null)}
                  >
                    <TooltipTrigger render={button} />
                    <TooltipContent>
                      {t('Please read and check the agreements below first.')}
                    </TooltipContent>
                  </Tooltip>
                )
              }
            )}
          </div>
        </TooltipProvider>
      </div>

      <TelegramLoginDialog
        open={isTelegramDialogOpen}
        botName={status?.telegram_bot_name ?? ''}
        pending={isTelegramPending}
        onOpenChange={setIsTelegramDialogOpen}
        onAuthorization={handleTelegramAuthorization}
      />
    </>
  )
}
