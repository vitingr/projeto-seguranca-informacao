import type { ListScope } from '@/domain/ticket-query/types'
import type { TicketSummary } from '@/domain/tickets/types'

export interface TicketTableProps {
  /** The caption read by assistive technology ("Meus tickets"). */
  caption: string
  items: readonly TicketSummary[]
  /** The address of the list on screen, carried in `?de=` (AC-2.5). */
  listHref: string
  /** `queue` adds the "#" and "Usuário" columns (AC-4.2). */
  scope: ListScope
}

export interface TicketDateProps {
  iso: string
}
