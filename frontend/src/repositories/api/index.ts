// Browser client for the state-changing API calls. Every call goes to the
// frontend origin (/api/*, proxied to the backend), so the browser sends the
// session cookie and Sec-Fetch-Site: same-origin by itself.
//
// Rules (spec D-3, D-7):
// - every POST sends X-CSRF-Token;
// - a 403 refreshes the token and retries once (an expired session makes the
//   backend answer 403 on a POST, because CSRF runs before Authorize; the
//   retry then gets the 401);
// - a 401 anywhere but POST /api/login discards the token and goes to the
//   login page;
// - any other answer, 429 included, is returned as it is: the client never
//   retries it by itself (spec 001 AC-6.7).

import { networkError } from '@/domain/api-error'
import { LOGIN_PATH } from '@/domain/login-path'
import { loginUrl } from '@/domain/navigation'
import { CSRF_HEADER } from '@/lib/api/contract'
import type { TicketResponse, UserResponse } from '@/lib/api/contract/types'
import { csrfStore } from '@/lib/api/csrf'
import { ApiRequestError, readError, toApiError } from '@/lib/api/errors'
import { never } from '@/utils/helpers/never'

import type { ApiClient, ApiClientDeps, Result } from './types'

export const createApiClient = (deps: ApiClientDeps): ApiClient => {
  const send = async (path: string, token: string, body?: unknown) => {
    const headers: Record<string, string> = { [CSRF_HEADER]: token }
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    try {
      return await deps.fetch(path, {
        body: body === undefined ? undefined : JSON.stringify(body),
        cache: 'no-store',
        headers,
        method: 'POST'
      })
    } catch {
      throw new ApiRequestError(networkError())
    }
  }

  /** POST with the token, the single retry on 403 and the 401 rule. */
  const post = async (
    path: string,
    body: unknown,
    { unauthorizedIsAnswer = false } = {}
  ): Promise<Response> => {
    let res = await send(path, await deps.csrf.get(), body)
    if (res.status === 403) {
      res = await send(path, await deps.csrf.refresh(), body)
    }
    if (res.status === 401 && !unauthorizedIsAnswer) {
      deps.csrf.clear()
      deps.navigation.go(
        loginUrl(deps.navigation.currentPath(), { expired: true })
      )
      return never()
    }
    return res
  }

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
    createTicket: input =>
      run(async () => {
        const res = await post('/api/tickets', input)
        if (res.status !== 201)
          return { error: await readError(res), ok: false }
        return { data: await json<TicketResponse>(res), ok: true }
      }),

    login: input =>
      run(async () => {
        const res = await post('/api/login', input, {
          unauthorizedIsAnswer: true
        })
        if (res.status !== 200)
          return { error: await readError(res), ok: false }
        const user = await json<UserResponse>(res)
        // The session and its token changed (002 D-4).
        await deps.csrf.refresh()
        return { data: user, ok: true }
      }),

    logout: () =>
      run(async () => {
        const res = await post('/api/logout', undefined, {
          unauthorizedIsAnswer: true
        })
        // 401: the session had already ended, which is the goal anyway.
        if (res.status !== 204 && res.status !== 401) {
          return { error: await readError(res), ok: false }
        }
        deps.csrf.clear()
        deps.navigation.replace(LOGIN_PATH)
        return never()
      }),

    register: input =>
      run(async () => {
        const res = await post('/api/cadastro', input)
        if (res.status !== 201)
          return { error: await readError(res), ok: false }
        return { data: await json<UserResponse>(res), ok: true }
      })
  }
}

/** The browser's client. `window` is read only when a call runs. */
export const api = createApiClient({
  csrf: csrfStore,
  fetch: (...args) => fetch(...args),
  navigation: {
    currentPath: () => window.location.pathname,
    go: url => window.location.assign(url),
    replace: url => window.location.replace(url)
  }
})
