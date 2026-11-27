import type { ReactNode } from 'react'

import type { TicketDetail } from '@/domain/tickets/types'

export interface TicketHeaderProps {
  /** Admin only: the status action (spec 003 US-6, D-5). */
  actions?: ReactNode
  ticket: TicketDetail
  /** The id of the `h1`, which gets the focus when the page opens. */
  titleId: string
}

export interface WhenProps {
  iso: string
}

export interface HeaderFactProps {
  children: ReactNode
  label: string
}
