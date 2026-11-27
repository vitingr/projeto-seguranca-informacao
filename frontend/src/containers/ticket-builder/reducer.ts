import type { BuilderAction, BuilderState } from './types'

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
      return state.type === null
        ? { ...state, typeError: true }
        : { ...state, pendingType: null, step: 'detalhes', typeError: false }
    case 'continueFromDetails':
      return state.step === 'detalhes' ? { ...state, step: 'revisao' } : state
    case 'goToStep': {
      if (state.sending) return state
      const target = ORDER.indexOf(action.step as (typeof ORDER)[number])
      const current = ORDER.indexOf(state.step as (typeof ORDER)[number])
      if (target < 0 || current < 0 || target >= current) return state
      return { ...state, pendingType: null, sendError: null, step: action.step }
    }
    case 'sendStarted':
      return state.step === 'revisao' && !state.sending
        ? { ...state, sendError: null, sending: true }
        : state
    case 'sendSucceeded':
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
      return state.sending
        ? { ...state, sendError: action.message, sending: false }
        : state
    case 'restart':
      return initialState
  }
}
