import type { ApiError } from '@/domain/api-error/types'
import type {
  CadastroRequest,
  LoginRequest,
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

export interface ApiClient {
  login(input: LoginRequest): Promise<Result<UserResponse>>
  logout(): Promise<Result<void>>
  register(input: CadastroRequest): Promise<Result<UserResponse>>
}
