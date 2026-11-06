export const QUOTA_RESET_MODE_MONTHLY = 'monthly'
export const QUOTA_RESET_MODE_CYCLE = 'cycle'
export const QUOTA_RESET_MODE_NEVER = 'never'

/** 用户可见的策略取值（库中未设置按每月固定日清零展示） */
export type VisibleQuotaResetMode =
  | typeof QUOTA_RESET_MODE_MONTHLY
  | typeof QUOTA_RESET_MODE_CYCLE
  | typeof QUOTA_RESET_MODE_NEVER

export const QUOTA_RESET_MODE_OPTIONS: readonly VisibleQuotaResetMode[] = [
  QUOTA_RESET_MODE_MONTHLY,
  QUOTA_RESET_MODE_CYCLE,
  QUOTA_RESET_MODE_NEVER,
]

/** 未单独设置（''）时按每月固定日清零展示 */
export function displayQuotaResetMode(mode?: string): VisibleQuotaResetMode {
  if (
    mode === QUOTA_RESET_MODE_CYCLE ||
    mode === QUOTA_RESET_MODE_NEVER ||
    mode === QUOTA_RESET_MODE_MONTHLY
  ) {
    return mode
  }
  return QUOTA_RESET_MODE_MONTHLY
}

/** i18n label keys; components must render via t(labelKey) */
export const QUOTA_RESET_MODE_LABEL_KEYS: Record<VisibleQuotaResetMode, string> =
  {
    [QUOTA_RESET_MODE_MONTHLY]: 'Monthly fixed-day reset',
    [QUOTA_RESET_MODE_CYCLE]: 'Billing-cycle reset',
    [QUOTA_RESET_MODE_NEVER]: 'No reset (use until exhausted)',
  }

/** 额度生效时间不可编辑的原因；null 表示可编辑 */
export type QuotaEffectiveTimeLockReason = 'active-cycle' | 'no-quota' | null

/**
 * 周期清零用户的"额度生效时间"编辑判定，与后端校验同口径：
 * - active-cycle：周期进行中（锚点未到期），不得改生效时间
 * - no-quota：无剩余额度，须先分配额度
 */
export function getQuotaEffectiveTimeLock(params: {
  quota: number
  nextResetAt?: number
  now?: number
}): QuotaEffectiveTimeLockReason {
  const now = params.now ?? Math.floor(Date.now() / 1000)
  if ((params.nextResetAt ?? 0) > now) return 'active-cycle'
  if (params.quota <= 0) return 'no-quota'
  return null
}
