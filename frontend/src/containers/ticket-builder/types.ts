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
  pendingType: TicketType | null
  sendError: null | string
  sending: boolean
  step: Step
  ticket: CreatedTicket | null
  type: TicketType | null
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

export interface SendTiming {
  end: null | number
  start: number
}

export type TicketAnswer = Extract<
  Awaited<ReturnType<CreateTicket>>,
  { ok: true }
>['data']

export type FieldProblem =
  'tooLong' | 'tooShort' | Exclude<keyof typeof FIELD_MESSAGES, 'choice'>

export type TextKind = keyof typeof FIELD_LIMITS

export type TicketFormValues = Record<string, string>

export interface FieldSlot {
  field: TicketField
  wide: boolean
}

export interface HistoryEntry {
  index: number
  visit: string
}

export interface HistoryWindow {
  history: Pick<History, 'go' | 'pushState' | 'replaceState'>
  location: Pick<Location, 'pathname' | 'search'>
}

export type PopDecision =
  | { delta: number; kind: 'undo' }
  | { kind: 'adopt' }
  | { kind: 'apply' }
  | { kind: 'none' }

export interface StepHistory {
  back: (step: Step) => boolean
  current: () => Step
  pop: (state: unknown, locked: boolean) => null | Step
  push: (step: Step) => void
  restart: () => void
  sent: () => void
  start: () => void
}

export type TicketValuesByType = {
  [Type in TicketType]: z.output<(typeof ticketSchemas)[Type]>
}
