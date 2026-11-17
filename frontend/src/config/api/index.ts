// Backend origin for the /api rewrite (next.config.ts) and for server-side
// calls. Server-only by convention: never expose it as NEXT_PUBLIC_*. This
// module does not import `server-only` because next.config.ts uses it too.

export const DEFAULT_API_URL = 'https://localhost'

/**
 * Validates API_URL and returns its origin. It must be an absolute https
 * URL with no credentials, path, query or fragment. The error never repeats
 * the value, which could hold credentials.
 */
export function parseApiUrl(raw: string | undefined): string {
  const value = raw?.trim() || DEFAULT_API_URL
  let url: URL
  try {
    url = new URL(value)
  } catch {
    throw new Error('API_URL must be an absolute https URL')
  }
  if (url.protocol !== 'https:') {
    throw new Error('API_URL must use https')
  }
  if (url.username || url.password) {
    throw new Error('API_URL must not contain credentials')
  }
  if (url.pathname !== '/' || url.search || url.hash) {
    throw new Error(
      'API_URL must be an origin, with no path, query or fragment'
    )
  }
  return url.origin
}
