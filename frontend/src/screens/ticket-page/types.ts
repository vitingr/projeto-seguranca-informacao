import type { ReactNode } from 'react'

import type { SearchParams } from '@/domain/ticket-query/types'
import type { ThreadPage, TicketDetail } from '@/domain/tickets/types'

export interface TicketPageScreenProps {
  /** The raw `[id]` of the path; a malformed one is a missing ticket. */
  id: string
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
  thread: ThreadPage
}

export interface TicketNotFoundProps {
  /** The list the way back goes to. */
  backHref: string
  backLabel: string
}
