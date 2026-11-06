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
import type { PricingModel } from '@/features/pricing/types'

import type { UserModelRatioMap } from '../types'

// ============================================================================
// Per-user Model Ratio Overrides (pure helpers)
// ============================================================================

/** quota_type === 1 means per-request billing (no model ratio; override is a price coefficient) */
export const QUOTA_TYPE_PER_REQUEST = 1

/** billing_mode === 'tiered_expr' means tiered expression billing */
export const BILLING_MODE_TIERED_EXPR = 'tiered_expr'

/** One editable row in the per-user model ratio editor */
export interface UserModelRatioRow {
  modelName: string
  globalRatio: number
  /** False 表示全局未配置定价，globalRatio 只是兜底值，不应作为参考价展示 */
  configured: boolean
  /** Fixed per-request price (USD), only meaningful for per-request models */
  modelPrice: number
  quotaType: number
  billingMode: string
  overridden: boolean
}

/**
 * Build editable rows from the pricing catalog:
 * only models enabled for the given group are listed, sorted by name,
 * and flagged when the user already has a ratio override for them.
 */
export function buildModelRatioRows(
  models: PricingModel[],
  group: string,
  overrides: UserModelRatioMap
): UserModelRatioRow[] {
  return models
    .filter(
      (model) =>
        Array.isArray(model.enable_groups) && model.enable_groups.includes(group)
    )
    .map((model) => ({
      modelName: model.model_name,
      globalRatio: model.model_ratio,
      configured: model.ratio_configured !== false,
      modelPrice: model.model_price ?? 0,
      quotaType: model.quota_type,
      billingMode: model.billing_mode ?? '',
      overridden: Object.hasOwn(overrides, model.model_name),
    }))
    .sort((a, b) => a.modelName.localeCompare(b.modelName))
}

/** Result of parsing one ratio draft input */
export type RatioDraftParseResult =
  | { kind: 'inherit' }
  | { kind: 'valid'; ratio: number }
  | { kind: 'invalid' }

/**
 * Parse a discount input: empty string means "no discount (billed at the
 * global price)", otherwise the value must be a finite number in [0, 1)
 * (0 = free for this user, values >= 1 are not a discount).
 */
export function parseRatioDraft(value: string): RatioDraftParseResult {
  const trimmed = value.trim()
  if (trimmed === '') {
    return { kind: 'inherit' }
  }
  const num = Number(trimmed)
  if (!Number.isFinite(num) || num < 0 || num >= 1) {
    return { kind: 'invalid' }
  }
  return { kind: 'valid', ratio: num }
}

/**
 * Build the PUT payload from draft inputs (full replacement semantics:
 * empty drafts are dropped, i.e. they inherit the global ratio again).
 * Returns null when any draft is invalid so saving can be blocked.
 */
export function buildRatiosPayload(
  draft: Record<string, string>
): UserModelRatioMap | null {
  const payload: UserModelRatioMap = {}
  for (const [modelName, value] of Object.entries(draft)) {
    const parsed = parseRatioDraft(value)
    if (parsed.kind === 'invalid') {
      return null
    }
    if (parsed.kind === 'valid') {
      payload[modelName] = parsed.ratio
    }
  }
  return payload
}
