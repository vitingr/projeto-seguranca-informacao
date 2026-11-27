import type { ListQuery } from '@/domain/ticket-query/types'

export interface StatusFilterProps {
  /** The list query of the page; each link keeps the other filters. */
  query: ListQuery
}
