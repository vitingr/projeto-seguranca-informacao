import { describe, expect, it, vi } from 'vitest'

import { NETWORK, TICKET_CLOSED } from '@/domain/api-error'

import { sendReply } from './reply'

const ENTRY = {
  created_at: '2026-11-26T17:04:00Z',
  id: 77,
  kind: 'message',
  mine: true,
  side: 'author',
  text: 'Oi'
}

const failing = (status: number, code: string, requestId?: string) =>
  vi.fn(async () => ({
    error: { code, requestId, status },
    ok: false as const
  }))

const input = (raw: string) => ({ raw, scope: 'own' as const, ticketId: 7 })

const FAILURES: [number, string, string, boolean][] = [
  [
    422,
    'unprocessable_entity',
    'Dados inválidos. Revise a mensagem e tente novamente.',
    false
  ],
  [
    0,
    NETWORK,
    'Não foi possível conectar ao servidor. Tente novamente.',
    false
  ],
  [
    429,
    'too_many_requests',
    'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.',
    false
  ],
  [
    409,
    TICKET_CLOSED,
    'Este ticket foi fechado e não aceita novas mensagens.',
    true
  ],
  [404, 'not_found', 'Ticket não encontrado.', true]
]

describe('sendReply (spec 003 AC-3.5 to AC-3.7)', () => {
  it('sends the trimmed text and gives the stored entry id', async () => {
    const send = vi.fn(async () => ({ data: ENTRY, ok: true as const }))
    const outcome = await sendReply({ send }, input('  Oi\n'))
    expect(send).toHaveBeenCalledWith({
      scope: 'own',
      texto: 'Oi',
      ticketId: 7
    })
    expect(outcome).toEqual({ entryId: 77, kind: 'sent' })
  })

  it.each([
    ['', 'Preencha este campo.'],
    [' \n\t ', 'Preencha este campo.'],
    ['a'.repeat(2001), 'Use no máximo 2000 caracteres.'],
    ['a\u0000b', 'Remova os caracteres de controle do texto.']
  ])(
    'refuses %j with its field message, sending nothing',
    async (raw, message) => {
      const send = vi.fn()
      expect(await sendReply({ send }, input(raw))).toEqual({
        kind: 'invalid',
        message
      })
      expect(send).not.toHaveBeenCalled()
    }
  )

  it.each(FAILURES)(
    'a %i keeps the text with its message',
    async (status, code, message, refresh) => {
      const outcome = await sendReply(
        { send: failing(status, code) },
        input('Oi')
      )
      expect(outcome).toEqual({ kind: 'failed', message, refresh })
    }
  )

  it('a 500 shows the code of the request', async () => {
    const outcome = await sendReply(
      { send: failing(500, 'internal', 'req-1') },
      input('Oi')
    )
    expect(outcome).toEqual({
      kind: 'failed',
      message:
        'Ocorreu um erro inesperado. Tente novamente mais tarde. (código: req-1)',
      refresh: false
    })
  })
})
