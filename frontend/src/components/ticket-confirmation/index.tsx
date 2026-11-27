import { cn } from 'cn'
import { ArrowRightIcon, CircleCheckIcon } from 'lucide-react'
import Link from 'next/link'

import { DetailList, DetailRow } from '@/components/detail-list'
import { TicketTypeIcon } from '@/components/ticket-type-icon'
import { Badge } from '@/components/ui/shadcn/badge'
import { buttonVariants } from '@/components/ui/shadcn/button'
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
  type: 'Tipo',
  viewTicket: 'Ver ticket'
} as const

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
    <Link
      className={cn(buttonVariants({ variant: 'outline' }), 'w-full sm:w-fit')}
      href={`/usuarios/tickets/${ticket.id}`}
    >
      {CONFIRMATION_TEXTS.viewTicket}
      <ArrowRightIcon data-icon="inline-end" aria-hidden />
    </Link>
  </div>
)
