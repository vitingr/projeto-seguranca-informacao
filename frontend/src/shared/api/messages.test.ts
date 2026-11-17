import { describe, expect, it } from 'vitest'

import type { ApiError } from './errors'
import { MESSAGES, type MessageContext, messageFor } from './messages'

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
