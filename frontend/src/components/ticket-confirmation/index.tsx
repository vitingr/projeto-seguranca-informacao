import { CircleCheckIcon } from 'lucide-react'

import { DetailList, DetailRow } from '@/components/detail-list'
import { TicketTypeIcon } from '@/components/ticket-type-icon'
import { Badge } from '@/components/ui/shadcn/badge'
import { formatDateTime } from '@/domain/date-time'
import { TICKET_STATUS_LABELS, ticketTypeLabel } from '@/domain/tickets'

import type { TicketConfirmationProps } from './types'

export const CONFIRMATION_TEXTS = {
  createdAt: 'Criado em',
  heading: 'Ticket criado',
  intro: 'A equipe de infraestrutura recebeu o seu pedido.',
  number: 'Número',
  status: 'Status',
  title: 'Título',
  type: 'Tipo'
} as const

/**
 * The confirmation of a created ticket (spec AC-5.2): "Ticket criado", the
 * number as "#<id>", the type, the title, a muted status badge and the
 * creation date, in the grouped rows of the review. Values are plain text
 * (S-8).
 */
export const TicketConfirmation = ({
  headingId,
  headingRef,
  ticket
}: TicketConfirmationProps) => (
  <div className="flex flex-col gap-6">
    <div className="flex items-start gap-3">
      <CircleCheckIcon className="size-6 shrink-0" aria-hidden />
      <div className="flex flex-col gap-1">
        <h2
          className="text-base font-semibold outline-none"
          id={headingId}
          ref={headingRef}
          tabIndex={-1}
        >
          {CONFIRMATION_TEXTS.heading}
        </h2>
        <p className="text-muted-foreground text-sm">
          {CONFIRMATION_TEXTS.intro}
        </p>
      </div>
    </div>
    <DetailList>
      <DetailRow label={CONFIRMATION_TEXTS.number}>{`#${ticket.id}`}</DetailRow>
      <DetailRow label={CONFIRMATION_TEXTS.type}>
        <span className="flex items-center gap-2">
          <TicketTypeIcon className="size-7" type={ticket.type} />
          {ticketTypeLabel(ticket.type)}
        </span>
      </DetailRow>
      <DetailRow label={CONFIRMATION_TEXTS.title} preserveLines>
        {ticket.title}
      </DetailRow>
      <DetailRow label={CONFIRMATION_TEXTS.status}>
        <Badge variant="secondary">{TICKET_STATUS_LABELS[ticket.status]}</Badge>
      </DetailRow>
      <DetailRow label={CONFIRMATION_TEXTS.createdAt}>
        {formatDateTime(ticket.createdAt) ?? '—'}
      </DetailRow>
    </DetailList>
  </div>
)
