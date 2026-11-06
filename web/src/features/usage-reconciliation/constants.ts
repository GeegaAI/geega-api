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

// 与后端 service/billing_usage.go 的 usage_billing_path 常量保持一致。
// 值为 i18n 键（与英文 fallback 同字面量），展示时经 t() 翻译。
const USAGE_BILLING_PATH_LABEL_KEYS: Record<string, string> = {
  local: 'Local estimation (no upstream usage)',
  upstream: 'Upstream usage (untagged semantic)',
  'billing-usage-openai': 'Upstream · OpenAI semantic',
  'billing-usage-openai-estimated': 'Upstream · OpenAI semantic',
  'billing-usage-anthropic': 'Upstream · Anthropic semantic',
  'billing-usage-anthropic-estimated': 'Upstream · Anthropic semantic',
  'billing-usage-gemini': 'Upstream · Gemini semantic',
  'billing-usage-gemini-estimated': 'Upstream · Gemini semantic',
  legacy: 'Legacy logs (no source tag)',
}

export function getUsageBillingPathLabelKey(path: string): string {
  return USAGE_BILLING_PATH_LABEL_KEYS[path] ?? path
}
