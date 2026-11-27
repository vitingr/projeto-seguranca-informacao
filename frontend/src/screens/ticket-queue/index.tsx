import { EmptyState } from '@/components/empty-state'
import { Pager } from '@/components/pager'
import { StatusFilter } from '@/components/status-filter'
import { StatusSummary } from '@/components/status-summary'
import { TicketTable } from '@/components/ticket-table'
import { LoadError } from '@/containers/load-error'
import { TicketListFilters } from '@/containers/ticket-list-filters'
import { MESSAGES } from '@/domain/messages'
import { LIST_PATHS, listQueryHref, readListQuery } from '@/domain/ticket-query'
import { loadErrorMessage, ticketCount } from '@/screens/user-profile'
import { loadQueue, loadQueueAuthors } from '@/services/tickets'

import type { TicketQueueProps } from './types'

export const QUEUE_TEXTS = {
  clearFilters: 'Limpar filtros',
  empty: 'Nenhum ticket na fila.',
  noMatch: 'Nenhum ticket com esses filtros.',
  title: 'Fila de tickets'
} as const

export const TicketQueue = async ({ searchParams }: TicketQueueProps) => {
  const query = readListQuery(searchParams, 'queue')
  const [queue, authors] = await Promise.all([
    loadQueue(query),
    loadQueueAuthors()
  ])
  const headingId = 'queue-title'
  const filtered =
    query.status !== null || query.type !== null || query.user !== null

  const body = (() => {
    if (queue.kind === 'error') {
      return <LoadError message={loadErrorMessage(queue.requestId)} />
    }
    if (queue.kind === 'forbidden') {
      return <p className="text-sm">{MESSAGES.forbiddenPage}</p>
    }
    const { items, pages, statusCounts, total } = queue.data
    const logins = authors.kind === 'ok' ? authors.data : []
    return (
      <>
        {statusCounts ? (
          <StatusSummary counts={statusCounts} query={query} />
        ) : null}
        <TicketListFilters
          logins={logins}
          query={query}
          statusFilter={<StatusFilter query={query} />}
        >
          {total === 0 ? (
            <EmptyState
              action={
                filtered
                  ? { href: LIST_PATHS.queue, label: QUEUE_TEXTS.clearFilters }
                  : undefined
              }
              text={filtered ? QUEUE_TEXTS.noMatch : QUEUE_TEXTS.empty}
            />
          ) : (
            <>
              {items.length > 0 ? (
                <TicketTable
                  caption={QUEUE_TEXTS.title}
                  items={items}
                  listHref={listQueryHref(query)}
                  scope="queue"
                />
              ) : null}
              <Pager pages={pages} query={query} />
            </>
          )}
        </TicketListFilters>
      </>
    )
  })()

  return (
    <section
      aria-labelledby={headingId}
      className="flex flex-col gap-4"
      data-testid="ticket-queue"
      id="fila"
    >
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-lg font-semibold" id={headingId}>
          {QUEUE_TEXTS.title}
        </h2>
        {queue.kind === 'ok' ? (
          <p
            className="text-muted-foreground text-sm"
            data-testid="ticket-count"
          >
            {ticketCount(queue.data.total)}
          </p>
        ) : null}
      </div>
      {body}
    </section>
  )
}
