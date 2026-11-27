import type { SearchParams } from '@/domain/ticket-query/types'

export interface TicketQueueProps {
  /** The `searchParams` of `/administradores`: filters and page (AC-4.4). */
  searchParams: SearchParams
}
