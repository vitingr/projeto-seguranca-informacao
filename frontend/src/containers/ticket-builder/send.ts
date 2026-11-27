import { messageFor } from '@/domain/messages'
import type { CreatedTicket, TicketInput } from '@/domain/tickets/types'

import type { CreateTicket, SendOutcome, TicketAnswer } from './types'

export const clock = (): number => performance.now()

export const toCreatedTicket = (answer: TicketAnswer): CreatedTicket => ({
  createdAt: answer.created_at,
  id: answer.id,
  status: answer.status,
  title: answer.title,
  type: answer.type
})

export const sendTicket = async (
  createTicket: CreateTicket,
  input: TicketInput
): Promise<SendOutcome> => {
  const result = await createTicket(input)
  return result.ok
    ? { ticket: toCreatedTicket(result.data), type: 'sendSucceeded' }
    : { message: messageFor(result.error, 'action'), type: 'sendFailed' }
}
