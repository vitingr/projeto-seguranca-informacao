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

/** How a field is validated and shown (spec section 4). */
export type TicketFieldKind = 'choice' | 'long' | 'optional' | 'short' | 'url'

export interface ChoiceOption {
  code: string
  /** A muted one-line description under the label. */
  description: string
  label: string
  /** Shown in discreet red ("Produção", "Crítica"). */
  risky?: boolean
}

export interface TicketField {
  /** A short example or help text under the label. */
  hint?: string
  key: string
  kind: TicketFieldKind
  label: string
  options?: readonly ChoiceOption[]
  /** A short example inside the empty control (plan 5.4). */
  placeholder?: string
  required: boolean
}

/** The request key of any field of a type. */
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

/** A ticket as the builder sends it: `tipo` and the `dados` of that type. */
export type TicketInput = {
  [Type in TicketType]: { dados: TicketDataByType[Type]; tipo: Type }
}[TicketType]

/** A field of a ticket as the review and the ticket page show it. */
export interface TicketDetailRow {
  key: string
  label: string
  /** A fixed choice shown in discreet red ("Produção", "Crítica"). */
  risky: boolean
  /** `null` for an optional field left empty ("Não informado"). */
  value: null | string
}

/** A created ticket, as the confirmation shows it. */
export interface CreatedTicket {
  createdAt: string
  id: number
  status: TicketStatus
  title: string
  type: TicketType
}
