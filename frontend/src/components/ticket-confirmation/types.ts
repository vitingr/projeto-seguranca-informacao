import type { Ref } from 'react'

import type { CreatedTicket } from '@/domain/tickets/types'

export interface TicketConfirmationProps {
  /** The id of the heading, which gets the focus. */
  headingId: string
  headingRef?: Ref<HTMLHeadingElement>
  ticket: CreatedTicket
}
