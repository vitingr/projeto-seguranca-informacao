import { describe, expect, it } from 'vitest'

import { builderReducer, initialState } from './reducer'
import type { BuilderAction, BuilderState } from './types'

const run = (actions: BuilderAction[], from: BuilderState = initialState) =>
  actions.reduce(builderReducer, from)

describe('type step', () => {
  it('starts on "tipo" with no type selected (AC-2.1)', () => {
    expect(initialState).toEqual({
      step: 'tipo',
      type: null,
      typeError: false
    })
  })

  it('stays on the step and shows the message with no type (AC-2.2)', () => {
    const state = run([{ type: 'continueFromType' }])
    expect(state.step).toBe('tipo')
    expect(state.typeError).toBe(true)
  })

  it('selecting a type clears the message', () => {
    const state = run([
      { type: 'continueFromType' },
      { ticketType: 'bug', type: 'selectType' }
    ])
    expect(state).toEqual({ step: 'tipo', type: 'bug', typeError: false })
  })

  it('continues to "detalhes" with a type (AC-2.3)', () => {
    const state = run([
      { ticketType: 'deploy', type: 'selectType' },
      { type: 'continueFromType' }
    ])
    expect(state.step).toBe('detalhes')
    expect(state.type).toBe('deploy')
  })
})

describe('goToStep', () => {
  const onDetails = run([
    { ticketType: 'outro', type: 'selectType' },
    { type: 'continueFromType' }
  ])

  it('goes back to a step already done', () => {
    expect(run([{ step: 'tipo', type: 'goToStep' }], onDetails).step).toBe(
      'tipo'
    )
  })

  it('never skips to a later step', () => {
    expect(run([{ step: 'revisao', type: 'goToStep' }], onDetails)).toBe(
      onDetails
    )
    expect(run([{ step: 'detalhes', type: 'goToStep' }])).toBe(initialState)
  })

  it('ignores the step it is on and unknown steps', () => {
    expect(run([{ step: 'detalhes', type: 'goToStep' }], onDetails)).toBe(
      onDetails
    )
    expect(run([{ step: 'confirmacao', type: 'goToStep' }], onDetails)).toBe(
      onDetails
    )
  })
})
