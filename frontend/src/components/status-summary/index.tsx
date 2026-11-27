import { cn } from 'cn'
import Link from 'next/link'

import { STATUS_ORDER } from '@/components/status-filter'
import { listQueryHref } from '@/domain/ticket-query'
import { TICKET_STATUS_LABELS } from '@/domain/tickets'

import type { StatusSummaryProps } from './types'

export const SUMMARY_LABEL = 'Tickets por status'

/**
 * The summary of the queue (spec 003 AC-4.5): "Aberto 4 · Em andamento 2 ·
 * Resolvido 9 · Fechado 1" over the whole queue, whatever the filters; each
 * item is a link that sets the status filter (back on page 1).
 */
export const StatusSummary = ({ counts, query }: StatusSummaryProps) => (
  <nav aria-label={SUMMARY_LABEL}>
    <ul className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 gap-y-1 text-sm">
      {STATUS_ORDER.map((status, index) => (
        <li className="flex items-center gap-1.5" key={status}>
          {index > 0 ? <span aria-hidden>·</span> : null}
          <Link
            className={cn(
              'focus-visible:ring-ring/50 rounded-sm underline-offset-4 outline-none hover:underline focus-visible:ring-3',
              query.status === status && 'text-foreground font-medium'
            )}
            aria-current={query.status === status ? 'page' : undefined}
            data-status={status}
            href={listQueryHref(query, { status })}
          >
            {TICKET_STATUS_LABELS[status]}{' '}
            <span className="text-foreground font-medium tabular-nums">
              {counts[status]}
            </span>
          </Link>
        </li>
      ))}
    </ul>
  </nav>
)
