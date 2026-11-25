import { describe, expect, it } from 'vitest'

import { builderReducer, initialState } from './reducer'
import type { BuilderAction, BuilderState } from './types'

const run = (actions: BuilderAction[], from: BuilderState = initialState) =>
  actions.reduce(builderReducer, from)

describe('type step', () => {
  it('starts on "tipo" with no type selected (AC-2.1)', () => {
    expect(initialState).toEqual({
      pendingType: null,
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
    expect(state).toEqual({
      pendingType: null,
      step: 'tipo',
      type: 'bug',
      typeError: false
    })
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

describe('changing the type (AC-2.4)', () => {
  const backWithData = run([
    { ticketType: 'bug', type: 'selectType' },
    { type: 'continueFromType' },
    { step: 'tipo', type: 'goToStep' }
  ])

  it('changes at once while the form has no data', () => {
    const state = run(
      [{ hasData: false, ticketType: 'outro', type: 'selectType' }],
      backWithData
    )
    expect(state.type).toBe('outro')
    expect(state.pendingType).toBeNull()
  })

  it('waits for confirmation while the form has data', () => {
    const state = run(
      [{ hasData: true, ticketType: 'outro', type: 'selectType' }],
      backWithData
    )
    expect(state.type).toBe('bug')
    expect(state.pendingType).toBe('outro')
  })

  it('discards only after "confirm"', () => {
    const state = run(
      [
        { hasData: true, ticketType: 'outro', type: 'selectType' },
        { type: 'confirmTypeChange' }
      ],
      backWithData
    )
    expect(state.type).toBe('outro')
    expect(state.pendingType).toBeNull()
  })

  it('keeps the type on "cancel"', () => {
    const state = run(
      [
        { hasData: true, ticketType: 'outro', type: 'selectType' },
        { type: 'cancelTypeChange' }
      ],
      backWithData
    )
    expect(state.type).toBe('bug')
    expect(state.pendingType).toBeNull()
  })

  it('picking the same type again asks nothing', () => {
    const state = run(
      [
        { hasData: true, ticketType: 'outro', type: 'selectType' },
        { hasData: true, ticketType: 'bug', type: 'selectType' }
      ],
      backWithData
    )
    expect(state.type).toBe('bug')
    expect(state.pendingType).toBeNull()
  })

  it('continuing keeps the current type and drops the pending one', () => {
    const state = run(
      [
        { hasData: true, ticketType: 'outro', type: 'selectType' },
        { type: 'continueFromType' }
      ],
      backWithData
    )
    expect(state).toMatchObject({
      pendingType: null,
      step: 'detalhes',
      type: 'bug'
    })
  })

  it('confirming with nothing pending changes nothing', () => {
    expect(run([{ type: 'confirmTypeChange' }], backWithData)).toBe(
      backWithData
    )
  })
})

describe('details step', () => {
  it('continues to "revisao" from "detalhes" (AC-3.6)', () => {
    const state = run([
      { ticketType: 'acesso', type: 'selectType' },
      { type: 'continueFromType' },
      { type: 'continueFromDetails' }
    ])
    expect(state.step).toBe('revisao')
  })

  it('ignores "continueFromDetails" on other steps', () => {
    expect(run([{ type: 'continueFromDetails' }])).toBe(initialState)
  })

  it('goes back from "revisao" to "detalhes" (AC-4.2)', () => {
    const state = run([
      { ticketType: 'acesso', type: 'selectType' },
      { type: 'continueFromType' },
      { type: 'continueFromDetails' },
      { step: 'detalhes', type: 'goToStep' }
    ])
    expect(state.step).toBe('detalhes')
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
