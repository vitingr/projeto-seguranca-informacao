import type { ReactNode } from 'react'

import type { TicketDetail } from '@/domain/tickets/types'

export interface TicketHeaderProps {
  actions?: ReactNode
  ticket: TicketDetail
  titleId: string
}

export interface WhenProps {
  iso: string
}

export interface HeaderFactProps {
  children: ReactNode
  label: string
}
