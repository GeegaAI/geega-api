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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import { QUOTA_TYPE_VALUES } from '../constants'
import { isDynamicPricingModel } from '../lib/dynamic-price'
import { getUserModelRatio } from '../lib/model-helpers'
import type { PricingModel } from '../types'

interface UserRatioBadgeProps {
  model: PricingModel
  className?: string
}

/** Trim trailing zeros while keeping up to 4 decimal places (e.g. "1.25"). */
function formatRatioLabel(value: number): string {
  return String(Number(value.toFixed(4)))
}

/**
 * Compact discount line shown directly above the model name on cards:
 * "8.5折 • 专属折扣". The discount is the ratio between the discounted and
 * the original price ×10 (Chinese 折 convention). Token-based models replace
 * the model ratio, so the ratio is override ÷ global; tiered / per-request
 * models multiply the original price by the coefficient itself.
 */
export function UserDiscountBadge(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const override = getUserModelRatio(props.model)

  if (override === undefined) return null
  // 未配置定价时基准价是兜底值，折扣/倍率展示没有意义。
  if (props.model.ratio_configured === false) return null

  let value: number
  if (isDynamicPricingModel(props.model)) {
    value = override
  } else if (props.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    value = override
  } else {
    value = props.model.model_ratio ? override / props.model.model_ratio : 1
  }

  const discount = String(Number((value * 10).toFixed(2)))

  return (
    <div className='truncate text-sm leading-tight font-medium text-rose-600 dark:text-rose-400'>
      {t('{{discount}} • Exclusive discount', { discount })}
    </div>
  )
}

/**
 * Marks models with a viewer-specific exclusive ratio.
 * Renders nothing when the model has no user model ratio override.
 */
export function UserRatioBadge(props: UserRatioBadgeProps) {
  const { t } = useTranslation()
  const override = getUserModelRatio(props.model)

  if (override === undefined) return null
  // 未配置定价时基准价是兜底值，折扣/倍率展示没有意义。
  if (props.model.ratio_configured === false) return null

  // 阶梯计费模型不使用模型倍率，专属值语义为结算系数
  let title: string
  if (isDynamicPricingModel(props.model)) {
    title = t('Your exclusive settlement coefficient {{ratio}}', {
      ratio: formatRatioLabel(override),
    })
  } else if (props.model.quota_type === QUOTA_TYPE_VALUES.REQUEST) {
    // 按次计费模型没有模型倍率，专属值语义为价格系数
    title = t('Your exclusive price coefficient {{ratio}}', {
      ratio: formatRatioLabel(override),
    })
  } else {
    title = t('Your exclusive ratio {{ratio}} (global {{global}})', {
      ratio: formatRatioLabel(override),
      global: formatRatioLabel(props.model.model_ratio),
    })
  }

  return (
    <Badge variant='warning' title={title} className={props.className}>
      {t('Exclusive Price')}
    </Badge>
  )
}
