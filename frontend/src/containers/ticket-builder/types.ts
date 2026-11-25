import type { z } from 'zod'

import type { FIELD_LIMITS } from '@/domain/tickets'
import type { TicketField, TicketType } from '@/domain/tickets/types'

import type { FIELD_MESSAGES, ticketSchemas } from './schemas'

export type Step = 'confirmacao' | 'detalhes' | 'revisao' | 'tipo'

export interface BuilderState {
  /** A different type picked while the form has data (AC-2.4). */
  pendingType: TicketType | null
  step: Step
  type: TicketType | null
  /** "Escolha um tipo de ticket." is shown (AC-2.2). */
  typeError: boolean
}

export type BuilderAction =
  | { hasData?: boolean; ticketType: TicketType; type: 'selectType' }
  | { step: Step; type: 'goToStep' }
  | { type: 'cancelTypeChange' }
  | { type: 'confirmTypeChange' }
  | { type: 'continueFromDetails' }
  | { type: 'continueFromType' }

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
