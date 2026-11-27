import type { ListQuery } from '@/domain/ticket-query/types'
import type { StatusCounts } from '@/domain/tickets/types'

export interface StatusSummaryProps {
  /** Tickets in each status over the whole queue (AC-4.5). */
  counts: StatusCounts
  query: ListQuery
}
