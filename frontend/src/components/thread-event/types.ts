import type { TicketStatus } from '@/domain/tickets/types'

export interface ThreadEventProps {
  /** "Você", "Suporte" or a login, in dark text. */
  actor: string
  now: string
  /** The opening line has no status; a status change has its new one. */
  status?: TicketStatus
  time: string
}
