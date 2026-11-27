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
  /** Path of the current page, sent back as `next` after a 401. */
  currentPath(): string
  /** Full navigation that adds a history entry. */
  go(url: string): void
  /** Full navigation that replaces the current history entry. */
  replace(url: string): void
}

export interface ApiClientDeps {
  csrf: CsrfStore
  fetch: typeof fetch
  navigation: Navigation
}

/** The `comum` routes (`own`) or the admin routes (`queue`) of a thread. */
export type ThreadScope = 'own' | 'queue'

export interface ApiClient {
  /**
   * PATCH /api/administradores/tickets/{id}/status. 200 → the new status;
   * a 409 is returned with the code `stale_status`.
   */
  changeStatus(input: {
    de: TicketStatus
    para: TicketStatus
    ticketId: number
  }): Promise<Result<StatusChangeResponse>>
  /** POST /api/tickets. 201 → the new ticket; any error is returned once. */
  createTicket(input: CreateTicketRequest): Promise<Result<TicketResponse>>
  /**
   * GET …/{id}/mensagens?antes=<entry id>: the older entries, as JSON to be
   * parsed by repositories/tickets (`ThreadPage` or `AdminThreadPage`).
   */
  loadThreadBefore(input: {
    antes: number
    scope: ThreadScope
    ticketId: number
  }): Promise<Result<unknown>>
  login(input: LoginRequest): Promise<Result<UserResponse>>
  logout(): Promise<Result<void>>
  register(input: CadastroRequest): Promise<Result<UserResponse>>
  /**
   * POST …/{id}/mensagens. 201 → the stored entry, as JSON to be parsed by
   * repositories/tickets; a 409 is returned with the code `ticket_closed`.
   */
  sendMessage(input: {
    scope: ThreadScope
    texto: string
    ticketId: number
  }): Promise<Result<unknown>>
}
