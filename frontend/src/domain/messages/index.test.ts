import { describe, expect, it } from 'vitest'

import type { ApiError } from '@/domain/api-error/types'

import { MESSAGES, messageFor } from '.'
import type { MessageContext } from './types'

const error = (code: string, extra: Partial<ApiError> = {}): ApiError => ({
  code,
  status: 400,
  ...extra
})

describe('messageFor', () => {
  it.each<[string, MessageContext, string]>([
    ['unauthorized', 'login', 'Login ou senha incorretos.'],
    ['unauthorized', 'action', 'Sua sessão expirou. Entre novamente.'],
    ['unauthorized', 'cadastro', 'Sua sessão expirou. Entre novamente.'],
    [
      'bad_request',
      'login',
      'Não foi possível enviar os dados. Recarregue a página e tente novamente.'
    ],
    ['forbidden', 'action', 'Você não tem permissão para realizar esta ação.'],
    ['conflict', 'cadastro', 'Este login já está em uso.'],
    [
      'unprocessable_entity',
      'cadastro',
      'Dados inválidos. Revise os campos e tente novamente.'
    ],
    [
      'network',
      'login',
      'Não foi possível conectar ao servidor. Tente novamente.'
    ]
  ])('%s in %s', (code, context, want) => {
    expect(messageFor(error(code), context)).toBe(want)
  })

  it.each<MessageContext>(['action', 'cadastro', 'login'])(
    'too_many_requests in %s',
    context => {
      const tooMany = error('too_many_requests', {
        message: 'texto do backend',
        requestId: 'R',
        status: 429
      })
      expect(messageFor(tooMany, context)).toBe(
        'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.'
      )
    }
  )

  it('adds the request id to the internal message', () => {
    expect(
      messageFor(error('internal', { requestId: 'RID123' }), 'action')
    ).toBe(
      'Ocorreu um erro inesperado. Tente novamente mais tarde. (código: RID123)'
    )
    expect(messageFor(error('internal'), 'action')).toBe(MESSAGES.internal)
  })

  it('uses the backend message for an unknown code', () => {
    expect(
      messageFor(error('not_found', { message: 'Não encontrado.' }), 'action')
    ).toBe('Não encontrado.')
  })

  it('falls back to the internal message for an unknown code without one', () => {
    expect(messageFor(error('teapot', { requestId: 'R' }), 'action')).toBe(
      `${MESSAGES.internal} (código: R)`
    )
  })

  it('never shows the backend message for a known code', () => {
    const backend = { message: 'texto do backend' }
    expect(messageFor(error('conflict', backend), 'cadastro')).not.toContain(
      'texto do backend'
    )
  })
})

describe('messageFor on a ticket page', () => {
  it.each<[string, MessageContext, string]>([
    [
      'unprocessable_entity',
      'reply',
      'Dados inválidos. Revise a mensagem e tente novamente.'
    ],
    [
      'ticket_closed',
      'reply',
      'Este ticket foi fechado e não aceita novas mensagens.'
    ],
    [
      'stale_status',
      'status',
      'O status deste ticket foi alterado por outra pessoa. A página foi atualizada; confira antes de tentar de novo.'
    ],
    ['not_found', 'reply', 'Ticket não encontrado.'],
    ['not_found', 'status', 'Ticket não encontrado.'],
    [
      'too_many_requests',
      'reply',
      'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.'
    ],
    ['unauthorized', 'status', 'Sua sessão expirou. Entre novamente.']
  ])('%s (%s)', (code, context, want) => {
    expect(messageFor(error(code), context)).toBe(want)
  })
})
