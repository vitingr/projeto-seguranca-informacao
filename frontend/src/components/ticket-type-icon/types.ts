import type { TicketType } from '@/domain/tickets/types'

export interface TicketTypeIconProps {
  className?: string
  /** A dark tile, for the selected type card. */
  selected?: boolean
  /** `null` shows a neutral file icon (no type chosen yet). */
  type: null | TicketType
}
