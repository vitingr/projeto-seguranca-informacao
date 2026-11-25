// Types of the backend API, written by hand from backend/api/openapi.json.
// index.test.ts checks them against the contract.

import type {
  CadastroRequest,
  CreateTicketRequest,
  LoginRequest,
  TicketDataByType,
  TicketResponse,
  UserResponse
} from './types'

export const ERROR_CODES = [
  'bad_request',
  'unauthorized',
  'forbidden',
  'not_found',
  'conflict',
  'unprocessable_entity',
  'too_many_requests',
  'internal'
] as const

export const CSRF_HEADER = 'X-CSRF-Token'

export const LOGIN_REQUEST_FIELDS = [
  'login',
  'senha'
] as const satisfies readonly (keyof LoginRequest)[]

export const CADASTRO_REQUEST_FIELDS = [
  'login',
  'senha',
  'papel'
] as const satisfies readonly (keyof CadastroRequest)[]

export const USER_RESPONSE_FIELDS = [
  'user_id',
  'login',
  'role'
] as const satisfies readonly (keyof UserResponse)[]

// Tickets (POST /api/tickets). Request keys are in Portuguese and response
// keys in English, like the rest of the API (plan P-1). These are the codes
// on the wire; the labels shown to the user live in the domain.

export const TICKET_TYPES = ['acesso', 'bug', 'deploy', 'outro'] as const

export const TICKET_STATUSES = [
  'aberto',
  'em_andamento',
  'resolvido',
  'fechado'
] as const

export const ACCESS_LEVELS = ['leitura', 'escrita', 'administrador'] as const

export const SEVERITIES = ['baixa', 'media', 'alta', 'critica'] as const

export const ENVIRONMENTS = ['dev', 'homolog', 'prod'] as const

export const CREATE_TICKET_REQUEST_FIELDS = [
  'tipo',
  'dados'
] as const satisfies readonly (keyof CreateTicketRequest)[]

/** The keys of `dados` for each `tipo`, in contract order. */
export const TICKET_DATA_FIELDS = {
  acesso: ['sistema', 'nivel_acesso', 'justificativa'],
  bug: ['sistema', 'descricao', 'passos', 'severidade'],
  deploy: ['projeto', 'repositorio', 'ambiente', 'observacoes'],
  outro: ['titulo', 'descricao']
} as const satisfies {
  [Type in keyof TicketDataByType]: readonly (keyof TicketDataByType[Type])[]
}

export const TICKET_RESPONSE_FIELDS = [
  'id',
  'type',
  'title',
  'status',
  'created_at',
  'updated_at'
] as const satisfies readonly (keyof TicketResponse)[]
