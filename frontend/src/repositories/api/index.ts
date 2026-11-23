// Browser client for the state-changing API calls. Every call goes to the
// frontend origin (/api/*, proxied to the backend), so the browser sends the
// session cookie and Sec-Fetch-Site: same-origin by itself.
//
// Rules (spec D-3, D-7, and 005 D-8 for PUT and DELETE):
// - every mutation (POST, PUT, DELETE) sends X-CSRF-Token;
// - a 403 refreshes the token and retries once (an expired session makes the
//   backend answer 403 on a POST, because CSRF runs before Authorize; the
//   retry then gets the 401);
// - a 401 anywhere but POST /api/login discards the token and goes to the
//   login page.

import { networkError } from '@/domain/api-error'
import { LOGIN_PATH } from '@/domain/login-path'
import { loginUrl } from '@/domain/navigation'
import { CSRF_HEADER } from '@/lib/api/contract'
import type {
  PostRequest,
  RemovalRequest,
  UserResponse
} from '@/lib/api/contract/types'
import { csrfStore } from '@/lib/api/csrf'
import { ApiRequestError, readError, toApiError } from '@/lib/api/errors'
import { never } from '@/utils/helpers/never'

import type { ApiClient, ApiClientDeps, MutationMethod, Result } from './types'

/** The path of one post; the ID is a positive integer from the API. */
const postPath = (id: number) => {
  if (!Number.isSafeInteger(id) || id <= 0) {
    throw new Error('refusing a malformed post id')
  }
  return `/api/posts/${id}`
}

export const createApiClient = (deps: ApiClientDeps): ApiClient => {
  const send = async (
    method: MutationMethod,
    path: string,
    token: string,
    body?: unknown
  ) => {
    const headers: Record<string, string> = { [CSRF_HEADER]: token }
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

  /** A mutation with the token, the single retry on 403 and the 401 rule. */
  const mutate = async (
    method: MutationMethod,
    path: string,
    body: unknown,
    { unauthorizedIsAnswer = false } = {}
  ): Promise<Response> => {
    let res = await send(method, path, await deps.csrf.get(), body)
    if (res.status === 403) {
      res = await send(method, path, await deps.csrf.refresh(), body)
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
    login: input =>
      run(async () => {
        const res = await mutate('POST', '/api/login', input, {
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
        const res = await mutate('POST', '/api/logout', undefined, {
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
        const res = await mutate('POST', '/api/cadastro', input)
        if (res.status !== 201)
          return { error: await readError(res), ok: false }
        return { data: await json<UserResponse>(res), ok: true }
      }),

    // The wall re-reads the posts from the server after each change
    // (router.refresh), so these calls return no data.
    createPost: input =>
      run(async () => {
        const body: PostRequest = { body: input.body, title: input.title }
        const res = await mutate('POST', '/api/posts', body)
        if (res.status !== 201)
          return { error: await readError(res), ok: false }
        return { data: undefined, ok: true }
      }),

    deletePost: id =>
      run(async () => {
        const res = await mutate('DELETE', postPath(id), undefined)
        if (res.status !== 204)
          return { error: await readError(res), ok: false }
        return { data: undefined, ok: true }
      }),

    // An admin removes any post with a reason (006 D-2). The backend checks
    // the role again and answers 403 to anyone else.
    removePost: (id, reason) =>
      run(async () => {
        const body: RemovalRequest = { reason }
        const res = await mutate('POST', `${postPath(id)}/removal`, body)
        if (res.status !== 204)
          return { error: await readError(res), ok: false }
        return { data: undefined, ok: true }
      }),

    updatePost: (id, input) =>
      run(async () => {
        const body: PostRequest = { body: input.body, title: input.title }
        const res = await mutate('PUT', postPath(id), body)
        if (res.status !== 200)
          return { error: await readError(res), ok: false }
        return { data: undefined, ok: true }
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
