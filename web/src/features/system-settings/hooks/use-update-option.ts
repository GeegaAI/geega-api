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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { toast } from 'sonner'

import { mapStatusDataToConfig } from '@/hooks/use-system-config'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { updateSystemOption } from '../api'
import type { UpdateOptionRequest } from '../types'

// 保存展示相关配置后立即拉取 /api/status 同步全局配置 store。
// 仅 invalidate status 查询不够：不挂 useStatus 的页面
// （如模型定价编辑页）会继续读到旧基准货币/展示类型
async function refreshStatusConfig(): Promise<void> {
  try {
    const response = await fetch('/api/status')
    if (!response.ok) return
    const payload = (await response.json()) as {
      success?: boolean
      data?: import('@/hooks/use-system-config').StatusApiResponse['data']
    }
    if (payload?.success && payload.data) {
      useSystemConfigStore
        .getState()
        .setConfig(mapStatusDataToConfig(payload.data))
    }
  } catch {
    /* 同步失败可忽略：页面挂载 useStatus 时会重新同步 */
  }
}

// 法律文档保存/发布后立即失效 status 缓存并重拉：
// legal_current_version 变更若等 5 分钟自然过期，
// 发布者本人登录会被版本不匹配反复提示
export async function invalidateLegalStatusCache(
  queryClient: ReturnType<typeof useQueryClient>
): Promise<void> {
  queryClient.invalidateQueries({ queryKey: ['status'] })
  try {
    window.localStorage.removeItem('status')
  } catch {
    /* empty */
  }
  await refreshStatusConfig()
}

// Configuration keys that require status refresh
const STATUS_RELATED_KEYS = new Set([
  'HeaderNavModules',
  'SidebarModulesAdmin',
  'Notice',
  'LogConsumeEnabled',
  'QuotaPerUnit',
  'USDExchangeRate',
  'DisplayInCurrencyEnabled',
  'DisplayTokenStatEnabled',
  'general_setting.quota_display_type',
  'general_setting.base_currency',
  'general_setting.custom_currency_symbol',
  'general_setting.custom_currency_exchange_rate',
  'oidc.display_name',
])

export function useUpdateOption() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: UpdateOptionRequest) => updateSystemOption(request),
    onSuccess: (data, variables) => {
      if (data.success) {
        // Always refresh system-options
        queryClient.invalidateQueries({ queryKey: ['system-options'] })

        // If updating frontend-display-related config, also refresh status
        if (STATUS_RELATED_KEYS.has(variables.key)) {
          queryClient.invalidateQueries({ queryKey: ['status'] })
          try {
            window.localStorage.removeItem('status')
          } catch {
            /* empty */
          }
          void refreshStatusConfig()
        }

        toast.success(i18next.t('Setting updated successfully'))
      } else {
        toast.error(data.message || i18next.t('Failed to update setting'))
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || i18next.t('Failed to update setting'))
    },
  })
}
