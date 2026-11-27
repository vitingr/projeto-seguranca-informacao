import type { ListScope } from '@/domain/ticket-query/types'
import type { TicketSummary } from '@/domain/tickets/types'

export interface TicketTableProps {
  caption: string
  items: readonly TicketSummary[]
  listHref: string
  scope: ListScope
}

export interface TicketDateProps {
  iso: string
}
