import type { BackendGet } from '@/lib/backend/types'

/**
 * What a page renders after a load: the data, the forbidden screen, or the
 * inline error with "Tentar novamente" (AC-1.10). A 401 and a missing
 * ticket never come back: they redirect to the login or render the
 * segment's not-found screen.
 */
export type LoadResult<T> =
  | { data: T; kind: 'ok' }
  | { kind: 'error'; requestId?: string }
  | { kind: 'forbidden' }

export interface TicketLoaderDeps {
  /** The session cookie of the request, as the browser sent it. */
  cookie(): Promise<string | undefined>
  get: BackendGet
  notFound(): never
  redirect(url: string): never
}
