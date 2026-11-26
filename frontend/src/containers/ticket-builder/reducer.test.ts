import { describe, expect, it } from 'vitest'

import { builderReducer, initialState } from './reducer'
import type { BuilderAction, BuilderState } from './types'

const run = (actions: BuilderAction[], from: BuilderState = initialState) =>
  actions.reduce(builderReducer, from)

describe('type step', () => {
  it('starts on "tipo" with no type selected (AC-2.1)', () => {
    expect(initialState).toEqual({
      pendingType: null,
      sendError: null,
      sending: false,
      step: 'tipo',
      ticket: null,
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
      ...initialState,
      type: 'bug'
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

describe('sending (US-5, US-6)', () => {
  const onReview = run([
    { ticketType: 'outro', type: 'selectType' },
    { type: 'continueFromType' },
    { type: 'continueFromDetails' }
  ])
  const ticket = {
    createdAt: '2026-11-25T14:30:00Z',
    id: 42,
    status: 'aberto' as const,
    title: 'Liberar porta',
    type: 'outro' as const
  }

  it('starts only once, from the review (AC-5.1)', () => {
    const sending = run([{ type: 'sendStarted' }], onReview)
    expect(sending.sending).toBe(true)
    expect(run([{ type: 'sendStarted' }], sending)).toBe(sending)
    expect(run([{ type: 'sendStarted' }])).toBe(initialState)
  })

  it('shows the confirmation only for a created ticket (AC-5.2)', () => {
    const state = run(
      [{ type: 'sendStarted' }, { ticket, type: 'sendSucceeded' }],
      onReview
    )
    expect(state).toMatchObject({
      sendError: null,
      sending: false,
      step: 'confirmacao',
      ticket
    })
    // An answer with no request pending changes nothing.
    expect(run([{ ticket, type: 'sendSucceeded' }], onReview)).toBe(onReview)
  })

  it('stays on the review with the message after an error (US-6)', () => {
    const state = run(
      [
        { type: 'sendStarted' },
        { message: 'Dados inválidos.', type: 'sendFailed' }
      ],
      onReview
    )
    expect(state).toMatchObject({
      sendError: 'Dados inválidos.',
      sending: false,
      step: 'revisao',
      type: 'outro'
    })
    // Sending again clears the message.
    expect(run([{ type: 'sendStarted' }], state).sendError).toBeNull()
  })

  it('does not leave the review while sending', () => {
    const sending = run([{ type: 'sendStarted' }], onReview)
    expect(run([{ step: 'detalhes', type: 'goToStep' }], sending)).toBe(sending)
  })

  it('cannot go back from the confirmation (AC-5.3)', () => {
    const done = run(
      [{ type: 'sendStarted' }, { ticket, type: 'sendSucceeded' }],
      onReview
    )
    expect(run([{ step: 'revisao', type: 'goToStep' }], done)).toBe(done)
  })

  it('"Criar outro ticket" starts over with nothing (AC-5.4)', () => {
    const done = run(
      [{ type: 'sendStarted' }, { ticket, type: 'sendSucceeded' }],
      onReview
    )
    expect(run([{ type: 'restart' }], done)).toEqual(initialState)
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
