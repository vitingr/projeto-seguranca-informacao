import { EmptyState } from '@/components/empty-state'
import { Forbidden } from '@/components/forbidden'
import { Pager } from '@/components/pager'
import { StatusFilter } from '@/components/status-filter'
import { TicketTable } from '@/components/ticket-table'
import { UserProfile } from '@/components/user-profile'
import { AppHeader } from '@/containers/app-header'
import { LoadError } from '@/containers/load-error'
import { TicketListFilters } from '@/containers/ticket-list-filters'
import { MESSAGES } from '@/domain/messages'
import { LIST_PATHS, listQueryHref, readListQuery } from '@/domain/ticket-query'
import { requireArea, requireRole } from '@/services/session'
import { loadOwnTickets } from '@/services/tickets'

import type { MyTicketsProps, UserProfileScreenProps } from './types'

export const MY_TICKETS_TEXTS = {
  clearFilters: 'Limpar filtros',
  empty: 'Você ainda não criou nenhum ticket.',
  noMatch: 'Nenhum ticket com esses filtros.',
  title: 'Meus tickets'
} as const

/** "1 ticket", "N tickets" (AC-1.7). */
export const ticketCount = (total: number) =>
  total === 1 ? '1 ticket' : `${total} tickets`

/** The generic message of a failed load, with its code when known (AC-1.10). */
export const loadErrorMessage = (requestId?: string) =>
  requestId ? `${MESSAGES.internal} (código: ${requestId})` : MESSAGES.network

/**
 * "Meus tickets" (spec 003 US-1): the count, the filters, the table and the
 * pages of the session user's own tickets, as the server answered them at
 * load time (BR-9). A failed load shows only the inline error.
 */
const MyTickets = ({ query, result }: MyTicketsProps) => {
  const headingId = 'my-tickets-title'
  const filtered = query.status !== null || query.type !== null
  const body = (() => {
    if (result.kind === 'error') {
      return <LoadError message={loadErrorMessage(result.requestId)} />
    }
    if (result.kind === 'forbidden') {
      return <p className="text-sm">{MESSAGES.forbiddenPage}</p>
    }
    const { items, pages, total } = result.data
    // The "Novo ticket" action of the page (profile, AC-1.1) is the next
    // step: a second link with the same name would only repeat it.
    if (total === 0 && !filtered) {
      return <EmptyState text={MY_TICKETS_TEXTS.empty} />
    }
    return (
      <TicketListFilters
        query={query}
        statusFilter={<StatusFilter query={query} />}
      >
        {total === 0 ? (
          <EmptyState
            action={{
              href: LIST_PATHS.own,
              label: MY_TICKETS_TEXTS.clearFilters
            }}
            text={MY_TICKETS_TEXTS.noMatch}
          />
        ) : (
          <>
            {items.length > 0 ? (
              <TicketTable
                caption={MY_TICKETS_TEXTS.title}
                items={items}
                listHref={listQueryHref(query)}
                scope="own"
              />
            ) : null}
            <Pager pages={pages} query={query} />
          </>
        )}
      </TicketListFilters>
    )
  })()

  return (
    <section
      aria-labelledby={headingId}
      className="flex flex-col gap-4"
      data-testid="my-tickets"
      id="meus-tickets"
    >
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-lg font-semibold" id={headingId}>
          {MY_TICKETS_TEXTS.title}
        </h2>
        {result.kind === 'ok' ? (
          <p
            className="text-muted-foreground text-sm"
            data-testid="ticket-count"
          >
            {ticketCount(result.data.total)}
          </p>
        ) : null}
      </div>
      {body}
    </section>
  )
}

/**
 * `/usuarios` (spec US-7, 003 US-1): the same guard as the other role areas
 * (the page role, then the backend's own answer on GET /api/usuarios), with
 * the profile of the session user and, under it, "Meus tickets". The data
 * comes from GET /api/me and GET /api/tickets, never from the URL or the
 * browser storage (AC-7.5); the address only holds the filter codes.
 */
export const UserProfileScreen = async ({
  searchParams
}: UserProfileScreenProps) => {
  const { allowed, user } = await requireRole('/usuarios')
  const area = allowed ? await requireArea('/api/usuarios', '/usuarios') : null
  const query = readListQuery(searchParams, 'own')
  const tickets = area?.kind === 'ok' ? await loadOwnTickets(query) : null

  return (
    <>
      <AppHeader role={user.role} />
      {area?.kind === 'ok' && tickets ? (
        <UserProfile
          area={area.area}
          createdAt={user.createdAt}
          login={user.login}
        >
          <MyTickets query={query} result={tickets} />
        </UserProfile>
      ) : (
        <Forbidden role={user.role} />
      )}
    </>
  )
}
