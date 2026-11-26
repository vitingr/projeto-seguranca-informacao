import type { TicketType } from '@/domain/tickets/types'

export interface TicketPreviewProps {
  /** Labels of the chosen options (severity, environment, access level). */
  badges?: readonly string[]
  /** The title the ticket will have, or `null` while it cannot be built. */
  title: null | string
  type: null | TicketType
  /** The label of the type, or `null` before a type is chosen. */
  typeLabel: null | string
}
