import { StatusBadge } from '@/components/status-badge'
import { TicketTypeIcon } from '@/components/ticket-type-icon'
import { formatDateTime } from '@/domain/date-time'
import { ticketTypeLabel } from '@/domain/tickets'

import type { HeaderFactProps, TicketHeaderProps, WhenProps } from './types'

export const HEADER_TEXTS = {
  author: 'Usuário',
  createdAt: 'Criado em',
  updatedAt: 'Atualizado em'
} as const

/** A date with its instant in `dateTime`, or a dash for a broken value. */
const When = ({ iso }: WhenProps) => {
  const text = formatDateTime(iso) ?? '—'
  return (
    <time dateTime={iso} title={text}>
      {text}
    </time>
  )
}

const HeaderFact = ({ children, label }: HeaderFactProps) => (
  <div className="flex items-baseline gap-1.5">
    <dt className="text-muted-foreground">{label}:</dt>
    <dd className="text-foreground min-w-0 break-all">{children}</dd>
  </div>
)

/**
 * The header of a ticket page (spec 003 AC-2.1, AC-5.1): "#<id>", the title
 * as the page's `h1`, the status badge, the type and both dates; the admin
 * page adds "Usuário: <login>" and the status action. Every value is plain
 * text (S-10).
 */
export const TicketHeader = ({
  actions,
  ticket,
  titleId
}: TicketHeaderProps) => (
  <header className="flex flex-col gap-3" data-testid="ticket-header">
    <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex min-w-0 flex-col gap-1.5">
        <p className="text-muted-foreground text-sm">{`#${ticket.id}`}</p>
        {/* Focused by the page on open, never by the Tab key (tabIndex -1),
            so it shows no ring; every control keeps its visible focus. */}
        <h1
          className="text-2xl font-semibold break-words outline-none"
          id={titleId}
          tabIndex={-1}
        >
          {ticket.title}
        </h1>
      </div>
      {actions ? <div className="flex shrink-0 gap-2">{actions}</div> : null}
    </div>
    <dl className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
      <div className="flex items-center">
        <dt className="sr-only">Status</dt>
        <dd>
          <StatusBadge status={ticket.status} />
        </dd>
      </div>
      <div className="flex items-center gap-2">
        <dt className="sr-only">Tipo</dt>
        <dd className="flex items-center gap-2">
          <TicketTypeIcon className="size-6" type={ticket.type} />
          {ticketTypeLabel(ticket.type)}
        </dd>
      </div>
      {ticket.authorLogin === undefined ? null : (
        <HeaderFact label={HEADER_TEXTS.author}>
          {ticket.authorLogin}
        </HeaderFact>
      )}
      <HeaderFact label={HEADER_TEXTS.createdAt}>
        <When iso={ticket.createdAt} />
      </HeaderFact>
      <HeaderFact label={HEADER_TEXTS.updatedAt}>
        <When iso={ticket.updatedAt} />
      </HeaderFact>
    </dl>
  </header>
)
