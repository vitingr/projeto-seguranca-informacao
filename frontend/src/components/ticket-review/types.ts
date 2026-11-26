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
  /** A fixed choice shown in discreet red ("Produção", "Crítica"). */
  risky?: boolean
  /** `null` for an optional field left empty ("Não informado"). */
  value: null | string
}

export interface TicketReviewProps {
  rows: readonly TicketReviewRow[]
  /** The title the ticket will get (BR-3). */
  title: string
  type: TicketType
  typeLabel: string
}
