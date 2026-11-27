import type { ReactNode } from 'react'

export interface TicketConversationProps {
  /** The thread entries, rendered by the server. */
  children: ReactNode
  /** What goes under the thread (the reply box, block 9). */
  footer?: ReactNode
  /** The messages of the thread (status lines are not counted). */
  messageCount: number
  /** When the server rendered the thread (ISO 8601): "Atualizado às". */
  renderedAt: string
}
