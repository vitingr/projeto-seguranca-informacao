import type { ReactNode } from 'react'

import type { TicketTab } from '@/domain/ticket-query/types'

export interface TicketTabsProps {
  /** The "Conversa" panel, rendered by the server. */
  conversation: ReactNode
  /** The "Detalhes" panel, rendered by the server. */
  details: ReactNode
  /** The tab read from the address on the server. */
  initialTab: TicketTab
  /** The id of the ticket title, focused when the page opens. */
  titleId: string
}
