// User-facing messages, in Portuguese, chosen by the `error` code (spec
// "Mensagens"). The backend `message` is used only for codes this table
// does not know.

import { NETWORK } from '@/domain/api-error'
import type { ApiError } from '@/domain/api-error/types'

import type { MessageContext } from './types'

export const MESSAGES = {
  badRequest:
    'Não foi possível enviar os dados. Recarregue a página e tente novamente.',
  conflict: 'Este login já está em uso.',
  forbiddenAction: 'Você não tem permissão para realizar esta ação.',
  forbiddenPage: 'Você não tem permissão para acessar esta página.',
  internal: 'Ocorreu um erro inesperado. Tente novamente mais tarde.',
  invalidCredentials: 'Login ou senha incorretos.',
  invalidData: 'Dados inválidos. Revise os campos e tente novamente.',
  network: 'Não foi possível conectar ao servidor. Tente novamente.',
  sessionExpired: 'Sua sessão expirou. Entre novamente.'
} as const

const withRequestId = (text: string, requestId?: string) =>
  requestId ? `${text} (código: ${requestId})` : text

export const messageFor = (
  error: ApiError,
  context: MessageContext
): string => {
  switch (error.code) {
    case NETWORK:
      return MESSAGES.network
    case 'bad_request':
      return MESSAGES.badRequest
    case 'unauthorized':
      return context === 'login'
        ? MESSAGES.invalidCredentials
        : MESSAGES.sessionExpired
    case 'forbidden':
      return MESSAGES.forbiddenAction
    case 'conflict':
      return MESSAGES.conflict
    case 'unprocessable_entity':
      return MESSAGES.invalidData
    case 'internal':
      return withRequestId(MESSAGES.internal, error.requestId)
    default:
      return error.message ?? withRequestId(MESSAGES.internal, error.requestId)
  }
}
