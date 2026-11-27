import type { ListQuery } from '@/domain/ticket-query/types'

export interface PagerProps {
  /** The total number of pages the backend answered (at least 1). */
  pages: number
  /** The list query of the page; `query.page` is the current page. */
  query: ListQuery
}
