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
import { describe, expect, test } from 'vitest'

import { PAYMENT_TYPES } from '../constants'
import { requestPaymentAmount } from './use-payment'

describe('payment amount routing', () => {
  test('uses the dedicated Stripe amount calculator for Stripe payments', async () => {
    const calls: string[] = []
    const amount = await requestPaymentAmount(120, PAYMENT_TYPES.STRIPE, {
      regular: async () => {
        calls.push('regular')
        return { success: true, data: '1' }
      },
      stripe: async (request) => {
        calls.push(`stripe:${request.amount}`)
        return { success: true, data: '18.75' }
      },
    })

    expect(amount).toBe(18.75)
    expect(calls).toEqual(['stripe:120'])
  })

  test('uses the regular amount calculator for non-Stripe payments', async () => {
    const calls: string[] = []
    const amount = await requestPaymentAmount(50, PAYMENT_TYPES.ALIPAY, {
      regular: async (request) => {
        calls.push(`regular:${request.amount}`)
        return { success: true, data: '7.5' }
      },
      stripe: async () => {
        calls.push('stripe')
        return { success: true, data: '2' }
      },
    })

    expect(amount).toBe(7.5)
    expect(calls).toEqual(['regular:50'])
  })
})
