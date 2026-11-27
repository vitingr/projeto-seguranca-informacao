import { STALE_STATUS, TICKET_CLOSED, networkError } from '@/domain/api-error'
import { LOGIN_PATH } from '@/domain/login-path'
import { loginUrl } from '@/domain/navigation'
import { CSRF_HEADER } from '@/lib/api/contract'
import type {
  StatusChangeResponse,
  TicketResponse,
  UserResponse
} from '@/lib/api/contract/types'
import { csrfStore } from '@/lib/api/csrf'
import { ApiRequestError, readError, toApiError } from '@/lib/api/errors'
import { never } from '@/utils/helpers/never'

import type { ApiClient, ApiClientDeps, Result, ThreadScope } from './types'

export const createApiClient = (deps: ApiClientDeps): ApiClient => {
  const send = async (
    method: string,
    path: string,
    token?: string,
    body?: unknown
  ) => {
    const headers: Record<string, string> = {}
    if (token !== undefined) headers[CSRF_HEADER] = token
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    try {
      return await deps.fetch(path, {
        body: body === undefined ? undefined : JSON.stringify(body),
        cache: 'no-store',
        headers,
        method
      })
    } catch {
      throw new ApiRequestError(networkError())
    }
  }

  const toLogin = <T>() => {
    deps.csrf.clear()
    deps.navigation.go(
      loginUrl(deps.navigation.currentPath(), { expired: true })
    )
    return never<T>()
  }

  // An expired session answers 403 on a POST (CSRF runs before auth): refresh the token, retry once.
  const write = async (
    method: 'PATCH' | 'POST',
    path: string,
    body: unknown,
    { unauthorizedIsAnswer = false } = {}
  ): Promise<Response> => {
    let res = await send(method, path, await deps.csrf.get(), body)
    if (res.status === 403) {
      res = await send(method, path, await deps.csrf.refresh(), body)
    }
    if (res.status === 401 && !unauthorizedIsAnswer) return toLogin<Response>()
    return res
  }

  const failure = async (res: Response, conflictCode?: string) => {
    const error = await readError(res)
    return res.status === 409 && conflictCode
      ? { ...error, code: conflictCode }
      : error
  }

  const threadPath = (scope: ThreadScope, ticketId: number) =>
    `${scope === 'queue' ? '/api/administradores/tickets' : '/api/tickets'}/${ticketId}/mensagens`

  const run = async <T>(call: () => Promise<Result<T>>): Promise<Result<T>> => {
    try {
      return await call()
    } catch (err) {
      return { error: toApiError(err), ok: false }
    }
  }

  const json = async <T>(res: Response): Promise<T> => {
    try {
      return (await res.json()) as T
    } catch {
      throw new ApiRequestError(networkError(res.status))
    }
  }

  return {
    changeStatus: ({ de, para, ticketId }) =>
      run(async () => {
        const res = await write(
          'PATCH',
          `/api/administradores/tickets/${ticketId}/status`,
          { de, para }
        )
        if (res.status !== 200) {
          return { error: await failure(res, STALE_STATUS), ok: false }
        }
        return { data: await json<StatusChangeResponse>(res), ok: true }
      }),

    createTicket: input =>
      run(async () => {
        const res = await write('POST', '/api/tickets', input)
        if (res.status !== 201)
          return { error: await readError(res), ok: false }
        return { data: await json<TicketResponse>(res), ok: true }
      }),

    loadThreadBefore: ({ antes, scope, ticketId }) =>
      run(async () => {
        const res = await send(
          'GET',
          `${threadPath(scope, ticketId)}?antes=${antes}`
        )
        if (res.status === 401) return toLogin<Result<unknown>>()
        if (res.status !== 200) return { error: await failure(res), ok: false }
        return { data: await json<unknown>(res), ok: true }
      }),

    login: input =>
      run(async () => {
        const res = await write('POST', '/api/login', input, {
          unauthorizedIsAnswer: true
        })
        if (res.status !== 200)
          return { error: await readError(res), ok: false }
        const user = await json<UserResponse>(res)
        await deps.csrf.refresh()
        return { data: user, ok: true }
      }),

    logout: () =>
      run(async () => {
        const res = await write('POST', '/api/logout', undefined, {
          unauthorizedIsAnswer: true
        })
        if (res.status !== 204 && res.status !== 401) {
          return { error: await readError(res), ok: false }
        }
        deps.csrf.clear()
        deps.navigation.replace(LOGIN_PATH)
        return never()
      }),

    register: input =>
      run(async () => {
        const res = await write('POST', '/api/cadastro', input)
        if (res.status !== 201)
          return { error: await readError(res), ok: false }
        return { data: await json<UserResponse>(res), ok: true }
      }),

    sendMessage: ({ scope, texto, ticketId }) =>
      run(async () => {
        const res = await write('POST', threadPath(scope, ticketId), { texto })
        if (res.status !== 201) {
          return { error: await failure(res, TICKET_CLOSED), ok: false }
        }
        return { data: await json<unknown>(res), ok: true }
      })
  }
}

export const api = createApiClient({
  csrf: csrfStore,
  fetch: (...args) => fetch(...args),
  navigation: {
    currentPath: () => window.location.pathname,
    go: url => window.location.assign(url),
    replace: url => window.location.replace(url)
  }
})
