import type { ReactNode } from 'react'

import type { ListScope, SearchParams } from '@/domain/ticket-query/types'
import type { ThreadPage, TicketDetail } from '@/domain/tickets/types'

export interface TicketPageScreenProps {
  actions?: (ticket: TicketDetail) => ReactNode
  id: string
  scope?: ListScope
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
  now: string
  scope: ListScope
  thread: ThreadPage
  ticket: TicketDetail
}

export interface TicketNotFoundProps {
  backHref: string
  backLabel: string
}
