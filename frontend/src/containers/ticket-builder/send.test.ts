import { describe, expect, it, vi } from 'vitest'

import { networkError } from '@/domain/api-error'
import type { ApiError } from '@/domain/api-error/types'
import { MESSAGES } from '@/domain/messages'
import type { TicketInput } from '@/domain/tickets/types'

import { sendTicket, toCreatedTicket } from './send'
import type { CreateTicket, TicketAnswer } from './types'

const INPUT: TicketInput = {
  dados: { descricao: 'Preciso da porta 8443.', titulo: 'Liberar porta' },
  tipo: 'outro'
}

const ANSWER: TicketAnswer = {
  created_at: '2026-11-25T14:30:00.123456Z',
  id: 42,
  status: 'aberto',
  title: 'Liberar porta',
  type: 'outro',
  updated_at: '2026-11-25T14:30:00.123456Z'
}

const failing = (error: ApiError): CreateTicket =>
  vi.fn(async () => ({ error, ok: false as const }))

describe('sendTicket', () => {
  it('sends the reviewed input and returns the created ticket', async () => {
    const createTicket = vi.fn(async () => ({
      data: ANSWER,
      ok: true as const
    }))
    const outcome = await sendTicket(createTicket, INPUT)
    expect(createTicket).toHaveBeenCalledExactlyOnceWith(INPUT)
    expect(outcome).toEqual({
      ticket: {
        createdAt: ANSWER.created_at,
        id: 42,
        status: 'aberto',
        title: 'Liberar porta',
        type: 'outro'
      },
      type: 'sendSucceeded'
    })
  })

  it.each([
    [
      'invalid data (422)',
      { code: 'unprocessable_entity', status: 422 },
      MESSAGES.invalidData
    ],
    [
      'too many requests (429)',
      { code: 'too_many_requests', status: 429 },
      MESSAGES.tooManyRequests
    ],
    [
      'lost permission (403)',
      { code: 'forbidden', status: 403 },
      MESSAGES.forbiddenAction
    ],
    ['network failure', networkError(), MESSAGES.network],
    [
      'server error with its code (500)',
      { code: 'internal', requestId: 'req-123', status: 500 },
      `${MESSAGES.internal} (código: req-123)`
    ]
  ])('turns %s into its message', async (_, error, message) => {
    const createTicket = failing(error)
    expect(await sendTicket(createTicket, INPUT)).toEqual({
      message,
      type: 'sendFailed'
    })
    expect(createTicket).toHaveBeenCalledTimes(1)
  })

  it('never shows the backend text or field contents', async () => {
    const outcome = await sendTicket(
      failing({
        code: 'unprocessable_entity',
        message: 'titulo: Liberar porta',
        status: 422
      }),
      INPUT
    )
    expect(outcome).toEqual({
      message: MESSAGES.invalidData,
      type: 'sendFailed'
    })
  })
})

describe('toCreatedTicket', () => {
  it('keeps only what the confirmation shows', () => {
    expect(toCreatedTicket(ANSWER)).not.toHaveProperty('updated_at')
  })
})
