import type { TicketStatus, TicketType } from '@/domain/tickets/types'

import type { LIST_PATHS } from '.'

/** "Meus tickets" (`own`) or the admin queue (`queue`). */
export type ListScope = keyof typeof LIST_PATHS

/** The filters and page of a list, as read from its address (AC-1.6). */
export interface ListQuery {
  /** From 1; page 1 is left out of the address. */
  page: number
  scope: ListScope
  /** `null` is "all". */
  status: null | TicketStatus
  type: null | TicketType
  /** The author's login; queue only. */
  user: null | string
}

export type ListQueryChange = Partial<
  Pick<ListQuery, 'page' | 'status' | 'type' | 'user'>
>

/** The `searchParams` of a Next page, or any record of query values. */
export type SearchParams = Record<string, string | string[] | undefined>
