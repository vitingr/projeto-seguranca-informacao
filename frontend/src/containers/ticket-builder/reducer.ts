import type { BuilderAction, BuilderState } from './types'

// The builder as a state machine (plan section 5.3): tipo → detalhes →
// revisao → confirmacao. Pure, so every transition is unit-tested.

export const initialState: BuilderState = {
  step: 'tipo',
  type: null,
  typeError: false
}

export const builderReducer = (
  state: BuilderState,
  action: BuilderAction
): BuilderState => {
  switch (action.type) {
    case 'selectType':
      return { ...state, type: action.ticketType, typeError: false }
    case 'continueFromType':
      if (state.step !== 'tipo') return state
      // AC-2.2: with no type, stay on the step and explain why.
      return state.type === null
        ? { ...state, typeError: true }
        : { ...state, step: 'detalhes', typeError: false }
    case 'goToStep': {
      // Only steps already done can be revisited (spec section 7).
      const order = ['tipo', 'detalhes', 'revisao'] as const
      const target = order.indexOf(action.step as (typeof order)[number])
      const current = order.indexOf(state.step as (typeof order)[number])
      if (target < 0 || current < 0 || target >= current) return state
      return { ...state, step: action.step }
    }
  }
}
