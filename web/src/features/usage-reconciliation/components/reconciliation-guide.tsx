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
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
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

const SINGLE_REQUEST_SQL = `SELECT id, created_at, model_name, channel_id,
       prompt_tokens, completion_tokens, upstream_request_id,
       other::jsonb -> 'upstream_usage' AS upstream_usage
FROM logs
WHERE type = 2 AND upstream_request_id = '<upstream_request_id>';`

export function ReconciliationGuide() {
  const { t } = useTranslation()

  const copySql = async () => {
    try {
      await navigator.clipboard.writeText(SINGLE_REQUEST_SQL)
      toast.success(t('Copied to clipboard'))
    } catch {
      toast.error(t('Failed to copy to clipboard'))
    }
  }

  return (
    <Card>
      <Collapsible>
        <CardHeader>
          <CollapsibleTrigger className="flex w-full items-center justify-between">
            <CardTitle className="text-base">{t('How the numbers are compared')}</CardTitle>
            <span className="text-muted-foreground text-xs">{t('Click to expand')}</span>
          </CollapsibleTrigger>
        </CardHeader>
        <CollapsibleContent>
          <CardContent className="space-y-2 text-sm">
            <p>{t('Anthropic upstream input = system input + cache read + cache creation. OpenAI and Gemini upstream input equals system input.')}</p>
            <p>{t('System output tokens are recorded verbatim from the upstream usage, so no separate upstream output is compared.')}</p>
            <p>{t('Rows marked Estimated or Local contain locally estimated tokens and cannot be reconciled with supplier bills.')}</p>
            <p>{t('Untagged rows without a recorded upstream semantic cannot be reconciled either; upstream and difference columns only include requests with a known semantic.')}</p>
            <p>{t('To verify a single request, look up its upstream_request_id in the supplier console and compare it with the upstream_usage snapshot in the log.')}</p>
            <Button variant="outline" size="sm" onClick={() => void copySql()}>
              {t('Copy single-request SQL')}
            </Button>
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  )
}
