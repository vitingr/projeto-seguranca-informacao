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

export const ticketCount = (total: number) =>
  total === 1 ? '1 ticket' : `${total} tickets`

export const loadErrorMessage = (requestId?: string) =>
  requestId ? `${MESSAGES.internal} (código: ${requestId})` : MESSAGES.network

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
