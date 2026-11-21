// The CSRF token lives only in this module's memory (spec D-3): never in
// Web Storage, IndexedDB, cookies or URLs. A page reload starts empty and
// the next state-changing request fetches a token again.

import { networkError } from '@/domain/api-error'
import type { CsrfResponse } from '@/lib/api/contract/types'
import { ApiRequestError, readError } from '@/lib/api/errors'

import type { CsrfStore } from './types'

export function createCsrfStore(fetchFn: typeof fetch): CsrfStore {
  let token: string | null = null
  let pending: Promise<string> | null = null

  async function load(): Promise<string> {
    let res: Response
    try {
      res = await fetchFn('/api/csrf', { cache: 'no-store' })
    } catch {
      throw new ApiRequestError(networkError())
    }
    if (!res.ok) throw new ApiRequestError(await readError(res))
    const body = (await res.json().catch(() => null)) as Partial<CsrfResponse>
    if (typeof body?.csrf_token !== 'string' || body.csrf_token === '') {
      throw new ApiRequestError(networkError(res.status))
    }
    return body.csrf_token
  }

  function refresh(): Promise<string> {
    token = null
    const request: Promise<string> = load().then(
      value => {
        if (pending === request) {
          token = value
          pending = null
        }
        return value
      },
      (err: unknown) => {
        if (pending === request) pending = null
        throw err
      }
    )
    pending = request
    return request
  }

  return {
    clear() {
      token = null
      pending = null
    },
    get() {
      if (token) return Promise.resolve(token)
      return pending ?? refresh()
    },
    refresh
  }
}

/** The browser's single store. */
export const csrfStore = createCsrfStore((...args) => fetch(...args))
