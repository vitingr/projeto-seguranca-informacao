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

/** What the builder writes in `history.state` for each of its entries. */
export interface HistoryEntry {
  /** The entry's position among the entries of this builder visit. */
  index: number
  /** Unique per visit (and per "Criar outro ticket"), so old entries are foreign. */
  visit: string
}

/** The part of `window` the step history needs (a fake one in tests). */
export interface HistoryWindow {
  history: Pick<History, 'go' | 'pushState' | 'replaceState'>
  location: Pick<Location, 'pathname' | 'search'>
}

/**
 * What to do with a `popstate` (history.ts): nothing, show the entry's
 * step, move the browser back to where it was, or take a foreign entry
 * over with the current step.
 */
export type PopDecision =
  | { delta: number; kind: 'undo' }
  | { kind: 'adopt' }
  | { kind: 'apply' }
  | { kind: 'none' }

export interface StepHistory {
  /** In-page "Voltar"/"Editar": moves the browser back to that step's entry. */
  back: (step: Step) => boolean
  /** The step of the entry the browser is on. */
  current: () => Step
  /** Handles a `popstate`; returns the step to show, if it changes. */
  pop: (state: unknown, locked: boolean) => null | Step
  /** A step reached by continuing: a new entry, so Back returns. */
  push: (step: Step) => void
  /** "Criar outro ticket": the current entry becomes the first step. */
  restart: () => void
  /** The ticket was created: back to the visit's first entry, as the confirmation. */
  sent: () => void
  /** Takes the entry the builder opened on as its first step. */
  start: () => void
}

/** Validated, trimmed values of each type. */
export type TicketValuesByType = {
  [Type in TicketType]: z.output<(typeof ticketSchemas)[Type]>
}
