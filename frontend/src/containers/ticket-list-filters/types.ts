import type { ReactNode } from 'react'

import type { ListQuery } from '@/domain/ticket-query/types'

export interface TicketListFiltersProps {
  /** The table, the empty state and the pager, marked busy while a filter loads. */
  children: ReactNode
  /** Queue only: the logins of the user filter (AC-4.4). */
  logins?: readonly string[]
  query: ListQuery
  /** The status links, shown first in the filter row. */
  statusFilter: ReactNode
}

/** The Select filters: type, and user in the queue. */
export type SelectFilter = 'type' | 'user'
