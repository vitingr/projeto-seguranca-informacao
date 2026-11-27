import { TicketTypeIcon } from '@/components/ticket-type-icon'
import { Badge } from '@/components/ui/shadcn/badge'

import type { TicketPreviewProps } from './types'

export const PREVIEW_TEXTS = {
  noTitle: 'Sem título ainda',
  noType: 'Tipo não escolhido'
} as const

export const TicketPreview = ({
  badges = [],
  title,
  type,
  typeLabel
}: TicketPreviewProps) => (
  <div className="flex items-start gap-3">
    <TicketTypeIcon className="size-10" type={type} />
    <div className="flex min-w-0 flex-1 flex-col gap-1">
      <p className="text-sm font-medium break-words">
        {title ?? PREVIEW_TEXTS.noTitle}
      </p>
      <p className="text-muted-foreground text-xs">
        {typeLabel ?? PREVIEW_TEXTS.noType}
      </p>
      {badges.length > 0 ? (
        <ul
          aria-label="Opções escolhidas"
          className="flex flex-wrap gap-1.5 pt-1"
        >
          {badges.map(badge => (
            <li key={badge}>
              <Badge variant="outline">{badge}</Badge>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  </div>
)
