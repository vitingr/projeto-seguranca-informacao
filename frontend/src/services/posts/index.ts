import 'server-only'

import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'

import { parseApiUrl } from '@/config/api'
import { env } from '@/config/env'
import { SESSION_COOKIE, createBackendGet } from '@/lib/backend'
import { loadPosts } from '@/repositories/posts'

const backendGet = createBackendGet(
  parseApiUrl({ raw: env.apiBaseUrl }),
  (...a) => fetch(...a)
)

/**
 * One page of the wall for the current user. Redirects to login if the
 * session ended after the page guard (requireRole) ran.
 */
export const requirePosts = async (page: number) => {
  const cookie = (await cookies()).get(SESSION_COOKIE)?.value ?? ''
  const result = await loadPosts(page, cookie, backendGet)
  if (result.kind === 'login') redirect(result.url)
  return result
}
