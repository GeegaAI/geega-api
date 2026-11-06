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

interface UserPricePrefixProps {
  /** Viewer's exclusive ratio/coefficient; renders nothing when absent or 1 */
  ratio: number | undefined
  /** Price string without the exclusive ratio applied */
  originalPrice: string
}

/**
 * Struck-through original price followed by an arrow leading to the caller's
 * exclusive (discounted) price: `¥2 →` + final price. Renders nothing when
 * the viewer has no effective override.
 */
export function UserPricePrefix(props: UserPricePrefixProps) {
  const { t } = useTranslation()
  const ratio = props.ratio

  if (
    ratio === undefined ||
    ratio === 1 ||
    !props.originalPrice ||
    props.originalPrice === '-'
  ) {
    return null
  }

  return (
    <>
      <span
        className='text-muted-foreground/60 line-through decoration-muted-foreground/50'
        title={t('Original price')}
      >
        {props.originalPrice}
      </span>
      <span
        aria-hidden
        className='text-rose-500/70 dark:text-rose-400/50'
      >
        →
      </span>
    </>
  )
}
