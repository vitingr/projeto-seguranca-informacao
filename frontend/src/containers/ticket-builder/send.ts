import { messageFor } from '@/domain/messages'
import type { CreatedTicket, TicketInput } from '@/domain/tickets/types'

import type { CreateTicket, SendOutcome, TicketAnswer } from './types'

/** The created ticket as the confirmation shows it (AC-5.2). */
export const toCreatedTicket = (answer: TicketAnswer): CreatedTicket => ({
  createdAt: answer.created_at,
  id: answer.id,
  status: answer.status,
  title: answer.title,
  type: answer.type
})

/**
 * Sends the reviewed ticket once and turns the answer into the next builder
 * action: the confirmation on 201, or the US-6 message otherwise (422, 429,
 * 403, network, 500 with its code). The client never retries a 429 and
 * goes to the login page by itself on 401 (AC-6.2).
 */
export const sendTicket = async (
  createTicket: CreateTicket,
  input: TicketInput
): Promise<SendOutcome> => {
  const result = await createTicket(input)
  return result.ok
    ? { ticket: toCreatedTicket(result.data), type: 'sendSucceeded' }
    : { message: messageFor(result.error, 'action'), type: 'sendFailed' }
}
