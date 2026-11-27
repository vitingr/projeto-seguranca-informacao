import type { TicketStatus, TicketType } from '@/domain/tickets/types'

import type { LIST_PATHS, TICKET_TABS } from '.'

export type TicketTab = (typeof TICKET_TABS)[number]

export type ListScope = keyof typeof LIST_PATHS

export interface ListQuery {
  page: number
  scope: ListScope
  status: null | TicketStatus
  type: null | TicketType
  user: null | string
}

export type ListQueryChange = Partial<
  Pick<ListQuery, 'page' | 'status' | 'type' | 'user'>
>

export type SearchParams = Record<string, string | string[] | undefined>
