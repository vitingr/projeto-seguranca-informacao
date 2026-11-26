import type { BuilderAction, BuilderState } from './types'

// The builder as a state machine (plan section 5.3): tipo → detalhes →
// revisao → confirmacao. Pure, so every transition is unit-tested. The
// typed values live in the form (react-hook-form), not here.

export const initialState: BuilderState = {
  pendingType: null,
  sendError: null,
  sending: false,
  step: 'tipo',
  ticket: null,
  type: null,
  typeError: false
}

const ORDER = ['tipo', 'detalhes', 'revisao'] as const

export const builderReducer = (
  state: BuilderState,
  action: BuilderAction
): BuilderState => {
  switch (action.type) {
    case 'selectType':
      if (action.ticketType === state.type) {
        return { ...state, pendingType: null, typeError: false }
      }
      // AC-2.4: with typed data, a different type waits for confirmation.
      return state.type !== null && action.hasData
        ? { ...state, pendingType: action.ticketType, typeError: false }
        : {
            ...state,
            pendingType: null,
            type: action.ticketType,
            typeError: false
          }
    case 'confirmTypeChange':
      return state.pendingType === null
        ? state
        : { ...state, pendingType: null, type: state.pendingType }
    case 'cancelTypeChange':
      return { ...state, pendingType: null }
    case 'continueFromType':
      if (state.step !== 'tipo') return state
      // AC-2.2: with no type, stay on the step and explain why.
      return state.type === null
        ? { ...state, typeError: true }
        : { ...state, pendingType: null, step: 'detalhes', typeError: false }
    case 'continueFromDetails':
      // AC-3.6: the container only sends this when the form is valid.
      return state.step === 'detalhes' ? { ...state, step: 'revisao' } : state
    case 'goToStep': {
      // Only steps already done can be revisited (spec section 7), never
      // while sending or after the ticket was created (AC-5.1, AC-5.3).
      if (state.sending) return state
      const target = ORDER.indexOf(action.step as (typeof ORDER)[number])
      const current = ORDER.indexOf(state.step as (typeof ORDER)[number])
      if (target < 0 || current < 0 || target >= current) return state
      return { ...state, pendingType: null, sendError: null, step: action.step }
    }
    case 'sendStarted':
      // AC-5.1: one request at a time, only from the review.
      return state.step === 'revisao' && !state.sending
        ? { ...state, sendError: null, sending: true }
        : state
    case 'sendSucceeded':
      // AC-5.2: the confirmation only exists for a created ticket.
      return state.sending
        ? {
            ...state,
            sendError: null,
            sending: false,
            step: 'confirmacao',
            ticket: action.ticket
          }
        : state
    case 'sendFailed':
      // US-6: stay on the review with the data kept.
      return state.sending
        ? { ...state, sendError: action.message, sending: false }
        : state
    case 'restart':
      // AC-5.4: back to the first step with nothing filled in.
      return initialState
  }
}
