import type { ApiError } from '@/domain/api-error/types'
import type {
  CadastroRequest,
  CreateTicketRequest,
  LoginRequest,
  StatusChangeResponse,
  TicketResponse,
  TicketStatus,
  UserResponse
} from '@/lib/api/contract/types'
import type { CsrfStore } from '@/lib/api/csrf/types'

export type Result<T> = { data: T; ok: true } | { error: ApiError; ok: false }

export interface Navigation {
  currentPath(): string
  go(url: string): void
  replace(url: string): void
}

export interface ApiClientDeps {
  csrf: CsrfStore
  fetch: typeof fetch
  navigation: Navigation
}

export type ThreadScope = 'own' | 'queue'

export interface ApiClient {
  changeStatus(input: {
    de: TicketStatus
    para: TicketStatus
    ticketId: number
  }): Promise<Result<StatusChangeResponse>>
  createTicket(input: CreateTicketRequest): Promise<Result<TicketResponse>>
  loadThreadBefore(input: {
    antes: number
    scope: ThreadScope
    ticketId: number
  }): Promise<Result<unknown>>
  login(input: LoginRequest): Promise<Result<UserResponse>>
  logout(): Promise<Result<void>>
  register(input: CadastroRequest): Promise<Result<UserResponse>>
  sendMessage(input: {
    scope: ThreadScope
    texto: string
    ticketId: number
  }): Promise<Result<unknown>>
}
