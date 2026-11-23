import type { ApiError } from '@/domain/api-error/types'
import type { PostInput } from '@/domain/post/types'
import type {
  CadastroRequest,
  LoginRequest,
  UserResponse
} from '@/lib/api/contract/types'
import type { CsrfStore } from '@/lib/api/csrf/types'

export type MutationMethod = 'DELETE' | 'POST' | 'PUT'

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
  createPost(input: PostInput): Promise<Result<void>>
  deletePost(id: number): Promise<Result<void>>
  login(input: LoginRequest): Promise<Result<UserResponse>>
  logout(): Promise<Result<void>>
  register(input: CadastroRequest): Promise<Result<UserResponse>>
  removePost(id: number, reason: string): Promise<Result<void>>
  updatePost(id: number, input: PostInput): Promise<Result<void>>
}
