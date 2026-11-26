import type { z } from 'zod'

import type { FIELD_LIMITS } from '@/domain/tickets'
import type {
  CreatedTicket,
  TicketField,
  TicketType
} from '@/domain/tickets/types'
import type { ApiClient } from '@/repositories/api/types'

import type { FIELD_MESSAGES, ticketSchemas } from './schemas'

export type Step = 'confirmacao' | 'detalhes' | 'revisao' | 'tipo'

export interface BuilderState {
  /** A different type picked while the form has data (AC-2.4). */
  pendingType: TicketType | null
  /** The US-6 message of the last failed send. */
  sendError: null | string
  /** The request is pending (AC-5.1). */
  sending: boolean
  step: Step
  /** The created ticket, on the confirmation (AC-5.2). */
  ticket: CreatedTicket | null
  type: TicketType | null
  /** "Escolha um tipo de ticket." is shown (AC-2.2). */
  typeError: boolean
}

export type SendOutcome =
  | { message: string; type: 'sendFailed' }
  | { ticket: CreatedTicket; type: 'sendSucceeded' }

export type BuilderAction =
  | SendOutcome
  | { hasData?: boolean; ticketType: TicketType; type: 'selectType' }
  | { step: Step; type: 'goToStep' }
  | { type: 'cancelTypeChange' }
  | { type: 'confirmTypeChange' }
  | { type: 'continueFromDetails' }
  | { type: 'continueFromType' }
  | { type: 'restart' }
  | { type: 'sendStarted' }

export type CreateTicket = ApiClient['createTicket']

/** When the send started and when the answer arrived (`performance.now`). */
export interface SendTiming {
  end: null | number
  start: number
}

/** The body of a 201 from `createTicket`. */
export type TicketAnswer = Extract<
  Awaited<ReturnType<CreateTicket>>,
  { ok: true }
>['data']

/** Why a field value is refused; maps to a message under the field. */
export type FieldProblem =
  'tooLong' | 'tooShort' | Exclude<keyof typeof FIELD_MESSAGES, 'choice'>

export type TextKind = keyof typeof FIELD_LIMITS

/** The values of a form as typed: every field is a string. */
export type TicketFormValues = Record<string, string>

/** A field in screen order; `wide` takes the whole row from `md`. */
export interface FieldSlot {
  field: TicketField
  wide: boolean
}

/** Validated, trimmed values of each type. */
export type TicketValuesByType = {
  [Type in TicketType]: z.output<(typeof ticketSchemas)[Type]>
}
