import type { BackendGet } from '@/lib/backend/types'

export type LoadResult<T> =
  | { data: T; kind: 'ok' }
  | { kind: 'error'; requestId?: string }
  | { kind: 'forbidden' }

export interface TicketLoaderDeps {
  cookie(): Promise<string | undefined>
  get: BackendGet
  notFound(): never
  redirect(url: string): never
}
