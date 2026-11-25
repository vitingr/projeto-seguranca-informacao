import type { z } from 'zod'

import type { FIELD_LIMITS } from '@/domain/tickets'
import type { TicketType } from '@/domain/tickets/types'

import type { FIELD_MESSAGES, ticketSchemas } from './schemas'

/** Why a field value is refused; maps to a message under the field. */
export type FieldProblem =
  'tooLong' | 'tooShort' | Exclude<keyof typeof FIELD_MESSAGES, 'choice'>

export type TextKind = keyof typeof FIELD_LIMITS

/** The values of a form as typed: every field is a string. */
export type TicketFormValues = Record<string, string>

/** Validated, trimmed values of each type. */
export type TicketValuesByType = {
  [Type in TicketType]: z.output<(typeof ticketSchemas)[Type]>
}
