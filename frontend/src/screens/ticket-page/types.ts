import type { ReactNode } from 'react'

import type { ListScope, SearchParams } from '@/domain/ticket-query/types'
import type { ThreadPage, TicketDetail } from '@/domain/tickets/types'

export interface TicketPageScreenProps {
  /** Admin page: the header actions of the loaded ticket (status, US-6). */
  actions?: (ticket: TicketDetail) => ReactNode
  /** The raw `[id]` of the path; a malformed one is a missing ticket. */
  id: string
  /** `own`: the author's page (default); `queue`: the admin's. */
  scope?: ListScope
  /** `aba` (the tab) and `de` (the list address to go back to). */
  searchParams: SearchParams
}

export interface TicketDetailsProps {
  ticket: TicketDetail
}

export interface DetailsGroupProps {
  children: ReactNode
  hint: string
  title: string
}

export interface ConversationProps {
  /** When the server rendered the page (ISO 8601). */
  now: string
  scope: ListScope
  thread: ThreadPage
  ticket: TicketDetail
}

export interface TicketNotFoundProps {
  /** The list the way back goes to. */
  backHref: string
  backLabel: string
}
