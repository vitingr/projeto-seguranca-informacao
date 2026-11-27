import type { Role } from '@/domain/roles/types'

import type {
  ACCESS_LEVELS,
  CSRF_HEADER,
  ENTRY_KINDS,
  ENTRY_SIDES,
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

// Ticket tracking (spec 003).

export type EntryKind = (typeof ENTRY_KINDS)[number]

export type EntrySide = (typeof ENTRY_SIDES)[number]

/** A row of "Meus tickets": never the fields of the type nor author data. */
export interface TicketSummary {
  created_at: string
  id: number
  status: TicketStatus
  title: string
  type: TicketType
  updated_at: string
}

/** A row of the queue: the summary plus the author's login. */
export interface AdminTicketSummary extends TicketSummary {
  author_login: string
}

export interface TicketPage {
  items: TicketSummary[]
  page: number
  page_size: number
  pages: number
  total: number
}

export type StatusCounts = Record<TicketStatus, number>

export interface QueuePage {
  items: AdminTicketSummary[]
  page: number
  page_size: number
  pages: number
  status_counts: StatusCounts
  total: number
}

export interface AuthorList {
  logins: string[]
}

/** The stored fields of a type: `observacoes` is always there, maybe null. */
export interface TicketDetailDataByType {
  acesso: AccessTicketData
  bug: BugTicketData
  deploy: Omit<DeployTicketData, 'observacoes'> & {
    observacoes: null | string
  }
  outro: OtherTicketData
}

interface TicketDetailBase {
  created_at: string
  id: number
  status: TicketStatus
  title: string
  updated_at: string
}

export type TicketDetail = {
  [Type in TicketType]: TicketDetailBase & {
    data: TicketDetailDataByType[Type]
    type: Type
  }
}[TicketType]

export type AdminTicketDetail = TicketDetail & { author_login: string }

interface ThreadEntryBase {
  created_at: string
  id: number
  mine: boolean
  side: EntrySide
}

export type ThreadEntry =
  | (ThreadEntryBase & {
      from: TicketStatus
      kind: 'status_change'
      to: TicketStatus
    })
  | (ThreadEntryBase & { kind: 'message'; text: string })

export type AdminThreadEntry = ThreadEntry & { actor_login: string }

export interface ThreadPage {
  entries: ThreadEntry[]
  has_more: boolean
}

export interface AdminThreadPage {
  entries: AdminThreadEntry[]
  has_more: boolean
}

export interface CreateMessageRequest {
  texto: string
}

export interface ChangeStatusRequest {
  de: TicketStatus
  para: TicketStatus
}

export interface StatusChangeResponse {
  id: number
  status: TicketStatus
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
