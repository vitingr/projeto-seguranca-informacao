import { cn } from 'cn'
import Link from 'next/link'

import { listQueryHref } from '@/domain/ticket-query'
import { TICKET_STATUS_LABELS } from '@/domain/tickets'
import type { TicketStatus } from '@/domain/tickets/types'

import type { StatusFilterProps } from './types'

export const STATUS_FILTER_TEXTS = {
  all: 'Todos',
  label: 'Filtrar por status'
} as const

/** The statuses in the order of their life (BR-3). */
export const STATUS_ORDER: readonly TicketStatus[] = [
  'aberto',
  'em_andamento',
  'resolvido',
  'fechado'
]

const OPTIONS: { label: string; status: null | TicketStatus }[] = [
  { label: STATUS_FILTER_TEXTS.all, status: null },
  ...STATUS_ORDER.map(status => ({
    label: TICKET_STATUS_LABELS[status],
    status
  }))
]

/**
 * The status filter of a list (spec 003 AC-1.5, plan P-13): a row of plain
 * links styled as the segmented control of `referencia-conversa.mp4`, the
 * active one on a gray fill and marked with `aria-current`. It works without
 * JavaScript, and each link goes back to the first page.
 */
export const StatusFilter = ({ query }: StatusFilterProps) => (
  <nav aria-label={STATUS_FILTER_TEXTS.label}>
    <ul className="flex flex-wrap items-center gap-1">
      {OPTIONS.map(({ label, status }) => {
        const active = query.status === status
        return (
          <li key={label}>
            <Link
              className={cn(
                'focus-visible:ring-ring/50 inline-flex h-8 items-center rounded-sm px-3 text-sm transition-colors outline-none focus-visible:ring-3',
                active
                  ? 'bg-muted text-foreground font-medium'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/60'
              )}
              aria-current={active ? 'page' : undefined}
              href={listQueryHref(query, { status })}
            >
              {label}
            </Link>
          </li>
        )
      })}
    </ul>
  </nav>
)
