import type { TicketType } from '@/domain/tickets/types'

export interface TicketPreviewProps {
  badges?: readonly string[]
  title: null | string
  type: null | TicketType
  typeLabel: null | string
}
