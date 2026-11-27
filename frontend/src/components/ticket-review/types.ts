import type { ReactNode } from 'react'

import type { TicketType } from '@/domain/tickets/types'

export interface ReviewSectionProps {
  children: ReactNode
  hint: string
  title: string
}

export interface TicketReviewRow {
  key: string
  label: string
  risky?: boolean
  value: null | string
}

export interface TicketReviewProps {
  rows: readonly TicketReviewRow[]
  title: string
  type: TicketType
  typeLabel: string
}
