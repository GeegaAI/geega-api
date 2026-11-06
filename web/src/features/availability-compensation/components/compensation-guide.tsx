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

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'

export function CompensationGuide() {
  const { t } = useTranslation()

  return (
    <Card>
      <Collapsible>
        <CardHeader>
          <CollapsibleTrigger className="flex w-full items-center justify-between">
            <CardTitle className="text-base">
              {t('How compensation is calculated')}
            </CardTitle>
            <span className="text-muted-foreground text-xs">
              {t('Click to expand')}
            </span>
          </CollapsibleTrigger>
        </CardHeader>
        <CollapsibleContent>
          <CardContent className="space-y-3 text-sm">
            <p>
              {t(
                'Availability = (total minutes − error minutes) ÷ total minutes × 100%. A minute counts as an error minute when the error rate within that minute reaches 50%.'
              )}
            </p>
            <p>
              {t(
                'Version 1 attributes all errors to the supplier. Version 2 only attributes server-side errors (5xx, 404, 408, 429, 529 and connection-level failures) and excludes client-side errors such as 400/401/403/413/418/422 from both the error count and the denominator.'
              )}
            </p>
            <div className="space-y-1">
              <p className="font-medium">{t('Compensation tiers')}</p>
              <ul className="text-muted-foreground list-disc space-y-0.5 pl-5">
                <li>{t('Availability > 99.9%: no compensation')}</li>
                <li>{t('99% ≤ availability ≤ 99.9%: 5%')}</li>
                <li>{t('95% ≤ availability < 99%: 8%')}</li>
                <li>{t('Availability < 95%: 10%')}</li>
              </ul>
            </div>
            <p>
              {t(
                'Availability is rounded to 3 decimals before the tier is applied, matching the settlement script.'
              )}
            </p>
            <p>
              {t(
                'The supplier is the channel name prefix before the first underscore; channels without a prefix are reported as 渠道{id}.'
              )}
            </p>
            <p>
              {t(
                'Only requests with a channel are included; requests without a channel (channel_id = 0) cannot be attributed to a supplier and are excluded.'
              )}
            </p>
            <p>
              {t(
                'The exported workbook contains the Notes, Industry Benchmark, Version 1, Version 2, Supplier Summary and Channel Mapping sheets.'
              )}
            </p>
            <p className="text-muted-foreground text-xs">
              {t(
                'Settlement figures come from the log store and reflect the data available at query time.'
              )}
            </p>
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  )
}
