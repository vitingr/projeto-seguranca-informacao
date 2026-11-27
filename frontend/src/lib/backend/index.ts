import type { BackendGet } from './types'

export const SESSION_COOKIE = '__Host-session'

// Only base32 tokens are forwarded, so a tampered cookie cannot inject headers
const SESSION_VALUE = /^[A-Z2-7]{26}$/

export const isSessionValue = (value: string | undefined): value is string =>
  value !== undefined && SESSION_VALUE.test(value)

export const createBackendGet = (
  apiUrl: string,
  fetchFn: typeof fetch
): BackendGet => {
  return async (path, session) => {
    if (!isSessionValue(session)) {
      throw new Error('refusing to forward a malformed session cookie')
    }
    const res = await fetchFn(`${apiUrl}${path}`, {
      cache: 'no-store',
      headers: { cookie: `${SESSION_COOKIE}=${session}` },
      redirect: 'error'
    })
    const body: unknown = await res.json().catch(() => null)
    return { body, status: res.status }
  }
}
