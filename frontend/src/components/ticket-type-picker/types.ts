import type { TICKET_TYPES } from '@/domain/tickets'
import type { TicketType } from '@/domain/tickets/types'

export interface TicketTypePickerProps {
  /** The message under the group ("Escolha um tipo de ticket."). */
  error?: string
  /** The id of the heading that names the group. */
  labelledBy: string
  onValueChange: (type: TicketType) => void
  options: typeof TICKET_TYPES
  value: TicketType | null
}
