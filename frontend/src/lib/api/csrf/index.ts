import { networkError } from '@/domain/api-error'
import type { CsrfResponse } from '@/lib/api/contract/types'
import { ApiRequestError, readError } from '@/lib/api/errors'

import type { CreaateCsrfStoreProps, CsrfStore } from './types'

export const createCsrfStore = ({
  fetchFn
}: CreaateCsrfStoreProps): CsrfStore => {
  let token: string | null = null
  let pending: Promise<string> | null = null

  const load = async (): Promise<string> => {
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

  const refresh = (): Promise<string> => {
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

export const csrfStore = createCsrfStore({
  fetchFn: (...args) => fetch(...args)
})
