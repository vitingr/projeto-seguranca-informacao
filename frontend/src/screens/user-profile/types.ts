import type { ListQuery, SearchParams } from '@/domain/ticket-query/types'
import type { TicketListPage } from '@/domain/tickets/types'
import type { LoadResult } from '@/services/tickets/types'

export interface UserProfileScreenProps {
  /** The `searchParams` of `/usuarios`: the filters and page (AC-1.6). */
  searchParams: SearchParams
}

export interface MyTicketsProps {
  query: ListQuery
  result: LoadResult<TicketListPage>
}
