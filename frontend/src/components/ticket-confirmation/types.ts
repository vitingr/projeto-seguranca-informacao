import type { Ref } from 'react'

import type { CreatedTicket } from '@/domain/tickets/types'

export interface TicketConfirmationProps {
  headingId: string
  headingRef?: Ref<HTMLHeadingElement>
  ticket: CreatedTicket
}
