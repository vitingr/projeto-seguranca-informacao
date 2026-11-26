import type { Role } from '@/domain/roles/types'

import type {
  ACCESS_LEVELS,
  CSRF_HEADER,
  ENVIRONMENTS,
  ERROR_CODES,
  SEVERITIES,
  TICKET_STATUSES,
  TICKET_TYPES
} from '.'

export type ErrorCode = (typeof ERROR_CODES)[number]

export type TicketType = (typeof TICKET_TYPES)[number]

export type TicketStatus = (typeof TICKET_STATUSES)[number]

export type AccessLevel = (typeof ACCESS_LEVELS)[number]

export type Severity = (typeof SEVERITIES)[number]

export type Environment = (typeof ENVIRONMENTS)[number]

export interface AccessTicketData {
  justificativa: string
  nivel_acesso: AccessLevel
  sistema: string
}

export interface BugTicketData {
  descricao: string
  passos: string
  severidade: Severity
  sistema: string
}

export interface DeployTicketData {
  ambiente: Environment
  /** Optional: absent, `null` and empty all mean no notes. */
  observacoes?: null | string
  projeto: string
  repositorio: string
}

export interface OtherTicketData {
  descricao: string
  titulo: string
}

export interface TicketDataByType {
  acesso: AccessTicketData
  bug: BugTicketData
  deploy: DeployTicketData
  outro: OtherTicketData
}

/** `tipo` and the `dados` of that type, with no other key. */
export type CreateTicketRequest = {
  [Type in TicketType]: { dados: TicketDataByType[Type]; tipo: Type }
}[TicketType]

export interface TicketResponse {
  created_at: string
  id: number
  status: TicketStatus
  title: string
  type: TicketType
  updated_at: string
}

export interface AreaResponse {
  area: string
}

export interface CadastroRequest {
  login: string
  papel: Role
  senha: string
}

export interface CsrfResponse {
  csrf_token: string
  header: typeof CSRF_HEADER
}

export interface ErrorResponse {
  error: string
  message: string
  request_id?: string
}

export interface LoginRequest {
  login: string
  senha: string
}

export interface UserResponse {
  /** When the user was registered (UTC); sent by GET /api/me only. */
  created_at?: string
  login?: string
  role: Role
  user_id: number
}
