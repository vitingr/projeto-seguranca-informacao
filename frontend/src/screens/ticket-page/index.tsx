import { cn } from 'cn'
import { ArrowLeftIcon } from 'lucide-react'
import Link from 'next/link'

import { DetailList, DetailRow } from '@/components/detail-list'
import { Forbidden } from '@/components/forbidden'
import { StatusBadge } from '@/components/status-badge'
import { Thread } from '@/components/thread'
import { TicketHeader } from '@/components/ticket-header'
import { TicketTypeIcon } from '@/components/ticket-type-icon'
import { buttonVariants } from '@/components/ui/shadcn/button'
import { AppHeader } from '@/containers/app-header'
import { LoadError } from '@/containers/load-error'
import { TicketConversation } from '@/containers/ticket-conversation'
import { TicketTabs } from '@/containers/ticket-tabs'
import { formatDateTime } from '@/domain/date-time'
import { MESSAGES } from '@/domain/messages'
import { backToListHref, readTicketTab } from '@/domain/ticket-query'
import {
  NOT_INFORMED,
  ticketDetailRows,
  ticketTypeLabel
} from '@/domain/tickets'
import { requireRole } from '@/services/session'
import { loadOwnThread, loadOwnTicket } from '@/services/tickets'

import type {
  ConversationProps,
  DetailsGroupProps,
  TicketDetailsProps,
  TicketNotFoundProps,
  TicketPageScreenProps
} from './types'

export const TICKET_PAGE_TEXTS = {
  backToMine: 'Voltar para meus tickets',
  createdAt: 'Criado em',
  details: 'Detalhes',
  detailsHint: 'Os campos exatamente como foram enviados.',
  newTicket: 'Novo ticket',
  number: 'Número',
  request: 'Pedido',
  replyPlaceholder: 'Escreva uma mensagem…',
  replyTray: 'Visível para a equipe de suporte',
  requestHint: 'O tipo, o título e o andamento do ticket.',
  status: 'Status',
  title: 'Título',
  type: 'Tipo',
  updatedAt: 'Atualizado em'
} as const

/** The generic message of a failed load, with its code when known. */
const loadErrorMessage = (requestId?: string) =>
  requestId ? `${MESSAGES.internal} (código: ${requestId})` : MESSAGES.network

const DetailsGroup = ({ children, hint, title }: DetailsGroupProps) => (
  <section className="flex flex-col gap-3">
    <div className="flex flex-col gap-0.5">
      <h2 className="text-sm font-semibold">{title}</h2>
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
    <DetailList>{children}</DetailList>
  </section>
)

/**
 * The "Detalhes" panel (spec 003 AC-2.2), in the grouped rows of
 * `settings.png`: "Pedido" (number, type, title, status, dates) and
 * "Detalhes" (every field of the type, as stored). Values are plain text with
 * their line breaks kept; an empty optional field shows "Não informado".
 */
const TicketDetails = ({ ticket }: TicketDetailsProps) => (
  <div className="flex flex-col gap-6">
    <DetailsGroup
      hint={TICKET_PAGE_TEXTS.requestHint}
      title={TICKET_PAGE_TEXTS.request}
    >
      <DetailRow label={TICKET_PAGE_TEXTS.number}>{`#${ticket.id}`}</DetailRow>
      <DetailRow label={TICKET_PAGE_TEXTS.type}>
        <span className="flex items-center gap-2">
          <TicketTypeIcon className="size-7" type={ticket.type} />
          {ticketTypeLabel(ticket.type)}
        </span>
      </DetailRow>
      <DetailRow label={TICKET_PAGE_TEXTS.title} preserveLines>
        {ticket.title}
      </DetailRow>
      <DetailRow label={TICKET_PAGE_TEXTS.status}>
        <StatusBadge status={ticket.status} />
      </DetailRow>
      <DetailRow label={TICKET_PAGE_TEXTS.createdAt}>
        {formatDateTime(ticket.createdAt) ?? '—'}
      </DetailRow>
      <DetailRow label={TICKET_PAGE_TEXTS.updatedAt}>
        {formatDateTime(ticket.updatedAt) ?? '—'}
      </DetailRow>
    </DetailsGroup>
    <DetailsGroup
      hint={TICKET_PAGE_TEXTS.detailsHint}
      title={TICKET_PAGE_TEXTS.details}
    >
      {ticketDetailRows(ticket.type, ticket.data).map(row => (
        <DetailRow key={row.key} label={row.label} preserveLines>
          {row.value === null ? (
            <span className="text-muted-foreground">{NOT_INFORMED}</span>
          ) : (
            <span className={cn(row.risky && 'text-destructive')}>
              {row.value}
            </span>
          )}
        </DetailRow>
      ))}
    </DetailsGroup>
  </div>
)

/**
 * The "Conversa" panel (spec 003 US-3): the thread as the server had it at
 * `now`, in the bounded panel with "Atualizado às · Atualizar".
 */
const Conversation = ({ now, thread, ticket }: ConversationProps) => (
  <TicketConversation
    messageCount={
      thread.entries.filter(entry => entry.kind === 'message').length
    }
    reply={{
      closed: ticket.status === 'fechado',
      closedAction: {
        href: '/usuarios/tickets/novo',
        label: TICKET_PAGE_TEXTS.newTicket
      },
      placeholder: TICKET_PAGE_TEXTS.replyPlaceholder,
      scope: 'own',
      ticketId: ticket.id,
      trayText: TICKET_PAGE_TEXTS.replyTray
    }}
    renderedAt={now}
  >
    <Thread
      entries={thread.entries}
      now={now}
      openedAt={ticket.createdAt}
      view="own"
    />
  </TicketConversation>
)

/** The link back to the list, with the filters the user had (AC-2.5). */
const BackLink = ({ backHref, backLabel }: TicketNotFoundProps) => (
  <Link
    className={cn(
      buttonVariants({ variant: 'ghost' }),
      'text-muted-foreground w-fit px-2'
    )}
    href={backHref}
  >
    <ArrowLeftIcon aria-hidden />
    {backLabel}
  </Link>
)

/**
 * "Ticket não encontrado." (spec 003 AC-2.4, AC-5.2): the same screen for a
 * ticket of someone else, a missing one and a malformed number, with the way
 * back to the list.
 */
export const TicketNotFound = ({
  backHref,
  backLabel
}: TicketNotFoundProps) => (
  <main className="mx-auto flex w-full max-w-7xl flex-col items-start gap-4 px-4 py-16 sm:px-6 lg:px-8">
    <h1 className="text-xl font-semibold" data-testid="ticket-not-found">
      {MESSAGES.ticketNotFound}
    </h1>
    <p className="text-muted-foreground text-sm">
      Confira o número do ticket ou volte para a lista.
    </p>
    <Link
      className={cn(buttonVariants({ variant: 'outline' }))}
      href={backHref}
    >
      {backLabel}
    </Link>
  </main>
)

/**
 * `/usuarios/tickets/<id>` (spec 003 US-2, plan 6.4): the comum guard, then
 * the ticket and its thread loaded in parallel on the server. A ticket that
 * is not the user's, a missing one or a malformed id renders the segment's
 * not-found screen (S-2); a failed load, the inline error. The tab and the
 * way back come from the address, checked (`?aba=`, `?de=`).
 */
export const TicketPageScreen = async ({
  id,
  searchParams
}: TicketPageScreenProps) => {
  const { allowed, user } = await requireRole(
    '/usuarios/tickets/[id]',
    `/usuarios/tickets/${id}`
  )
  if (!allowed) {
    return (
      <>
        <AppHeader role={user.role} />
        <Forbidden role={user.role} />
      </>
    )
  }

  const [ticket, thread] = await Promise.all([
    loadOwnTicket(id),
    loadOwnThread(id)
  ])
  const backHref = backToListHref(searchParams.de, 'own')
  const titleId = 'ticket-title'

  const body = (() => {
    if (ticket.kind === 'forbidden' || thread.kind === 'forbidden') {
      return <Forbidden role={user.role} />
    }
    if (ticket.kind === 'error' || thread.kind === 'error') {
      const requestId =
        (ticket.kind === 'error' ? ticket.requestId : undefined) ??
        (thread.kind === 'error' ? thread.requestId : undefined)
      return (
        <main className="mx-auto flex w-full max-w-7xl flex-col gap-4 px-4 py-10 sm:px-6 lg:px-8">
          <BackLink
            backHref={backHref}
            backLabel={TICKET_PAGE_TEXTS.backToMine}
          />
          <LoadError message={loadErrorMessage(requestId)} />
        </main>
      )
    }
    return (
      <main className="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
        <BackLink
          backHref={backHref}
          backLabel={TICKET_PAGE_TEXTS.backToMine}
        />
        <TicketHeader ticket={ticket.data} titleId={titleId} />
        <TicketTabs
          conversation={
            <Conversation
              now={new Date().toISOString()}
              thread={thread.data}
              ticket={ticket.data}
            />
          }
          details={<TicketDetails ticket={ticket.data} />}
          initialTab={readTicketTab(searchParams.aba)}
          titleId={titleId}
        />
      </main>
    )
  })()

  return (
    <>
      <AppHeader role={user.role} />
      {body}
    </>
  )
}
