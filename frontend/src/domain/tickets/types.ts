import type {
  ACCESS_LEVELS,
  ENVIRONMENTS,
  SEVERITIES,
  TICKET_FIELDS,
  TICKET_TYPES
} from '.'

export type TicketType = (typeof TICKET_TYPES)[number]['code']

export type AccessLevel = (typeof ACCESS_LEVELS)[number]['code']

export type Severity = (typeof SEVERITIES)[number]['code']

export type Environment = (typeof ENVIRONMENTS)[number]['code']

export type TicketStatus = 'aberto' | 'em_andamento' | 'fechado' | 'resolvido'

export type TicketFieldKind = 'choice' | 'long' | 'optional' | 'short' | 'url'

export interface ChoiceOption {
  code: string
  description: string
  label: string
  risky?: boolean
}

export interface TicketField {
  hint?: string
  key: string
  kind: TicketFieldKind
  label: string
  options?: readonly ChoiceOption[]
  placeholder?: string
  required: boolean
}

export type TicketFieldKey = (typeof TICKET_FIELDS)[TicketType][number]['key']

export interface AccessData {
  justificativa: string
  nivel_acesso: AccessLevel
  sistema: string
}

export interface BugData {
  descricao: string
  passos: string
  severidade: Severity
  sistema: string
}

export interface DeployData {
  ambiente: Environment
  observacoes?: string
  projeto: string
  repositorio: string
}

export interface OtherData {
  descricao: string
  titulo: string
}

export interface TicketDataByType {
  acesso: AccessData
  bug: BugData
  deploy: DeployData
  outro: OtherData
}

export type TicketInput = {
  [Type in TicketType]: { dados: TicketDataByType[Type]; tipo: Type }
}[TicketType]

export interface TicketSummary {
  authorLogin?: string
  createdAt: string
  id: number
  status: TicketStatus
  title: string
  type: TicketType
  updatedAt: string
}

export type StatusCounts = Record<TicketStatus, number>

export interface TicketListPage {
  items: TicketSummary[]
  page: number
  pages: number
  pageSize: number
  statusCounts?: StatusCounts
  total: number
}

export interface TicketDetail {
  authorLogin?: string
  createdAt: string
  data: Record<string, null | string>
  id: number
  status: TicketStatus
  title: string
  type: TicketType
  updatedAt: string
}

export type EntrySide = 'author' | 'support'

interface ThreadEntryBase {
  actorLogin?: string
  createdAt: string
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

export interface ThreadPage {
  entries: ThreadEntry[]
  hasMore: boolean
}

export interface StatusChange {
  id: number
  status: TicketStatus
  updatedAt: string
}

export interface TicketDetailRow {
  key: string
  label: string
  risky: boolean
  value: null | string
}

export interface CreatedTicket {
  createdAt: string
  id: number
  status: TicketStatus
  title: string
  type: TicketType
}
