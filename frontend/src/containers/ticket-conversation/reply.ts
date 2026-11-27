import { TICKET_CLOSED } from '@/domain/api-error'
import { messageFor } from '@/domain/messages'
import { checkMessage, trimField } from '@/domain/tickets'
import { parseThreadEntry } from '@/repositories/tickets'

import type { ReplyDeps, ReplyInput, ReplyOutcome } from './types'

/**
 * One reply (spec 003 AC-3.5 to AC-3.7): the message rule of the backend is
 * checked first, so an invalid text sends nothing and gets its field
 * message. A stored reply gives its entry id (for the fade-in); a failure
 * gives the AC-3.7 text and whether the page must be loaded again: a 409
 * (closed meanwhile) shows the closed line, a 404 the not-found screen. A
 * 401 never comes back: the client has already gone to the login page.
 */
export const sendReply = async (
  deps: ReplyDeps,
  { raw, scope, ticketId }: ReplyInput
): Promise<ReplyOutcome> => {
  const problem = checkMessage(raw)
  if (problem !== null) return { kind: 'invalid', message: problem }
  const result = await deps.send({ scope, texto: trimField(raw), ticketId })
  if (result.ok) {
    const entry = parseThreadEntry(result.data, scope)
    return { entryId: entry?.id ?? null, kind: 'sent' }
  }
  const { error } = result
  return {
    kind: 'failed',
    message: messageFor(error, 'reply'),
    refresh: error.code === TICKET_CLOSED || error.status === 404
  }
}
