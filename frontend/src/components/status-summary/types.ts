import type { ListQuery } from '@/domain/ticket-query/types'
import type { StatusCounts } from '@/domain/tickets/types'

export interface StatusSummaryProps {
  counts: StatusCounts
  query: ListQuery
}
