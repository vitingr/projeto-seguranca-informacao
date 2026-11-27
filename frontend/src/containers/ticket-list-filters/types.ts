import type { ReactNode } from 'react'

import type { ListQuery } from '@/domain/ticket-query/types'

export interface TicketListFiltersProps {
  children: ReactNode
  logins?: readonly string[]
  query: ListQuery
  statusFilter: ReactNode
}

export type SelectFilter = 'type' | 'user'
