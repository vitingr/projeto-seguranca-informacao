import { NETWORK, STALE_STATUS, TICKET_CLOSED } from '@/domain/api-error'
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
  invalidMessage: 'Dados inválidos. Revise a mensagem e tente novamente.',
  network: 'Não foi possível conectar ao servidor. Tente novamente.',
  sessionExpired: 'Sua sessão expirou. Entre novamente.',
  staleStatus:
    'O status deste ticket foi alterado por outra pessoa. A página foi atualizada; confira antes de tentar de novo.',
  ticketClosed: 'Este ticket foi fechado e não aceita novas mensagens.',
  ticketNotFound: 'Ticket não encontrado.',
  tooManyRequests:
    'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.'
} as const

const ticketMessageFor = (
  error: ApiError,
  context: 'reply' | 'status'
): null | string => {
  switch (error.code) {
    case TICKET_CLOSED:
      return MESSAGES.ticketClosed
    case STALE_STATUS:
      return MESSAGES.staleStatus
    case 'not_found':
      return MESSAGES.ticketNotFound
    case 'unprocessable_entity':
      return context === 'reply'
        ? MESSAGES.invalidMessage
        : MESSAGES.invalidData
    default:
      return null
  }
}

const withRequestId = (text: string, requestId?: string) =>
  requestId ? `${text} (código: ${requestId})` : text

export const messageFor = (
  error: ApiError,
  context: MessageContext
): string => {
  if (context === 'reply' || context === 'status') {
    const text = ticketMessageFor(error, context)
    if (text !== null) return text
  }
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
    case 'too_many_requests':
      return MESSAGES.tooManyRequests
    case 'internal':
      return withRequestId(MESSAGES.internal, error.requestId)
    default:
      return error.message ?? withRequestId(MESSAGES.internal, error.requestId)
  }
}
