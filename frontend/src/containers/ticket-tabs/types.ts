import type { ReactNode } from 'react'

import type { TicketTab } from '@/domain/ticket-query/types'

export interface TicketTabsProps {
  conversation: ReactNode
  details: ReactNode
  initialTab: TicketTab
  titleId: string
}
