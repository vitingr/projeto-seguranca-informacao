import { loginUrl } from '@/domain/navigation'
import type { Post, PostPage, PostsResult } from '@/domain/post/types'
import { isSessionValue } from '@/lib/backend'
import type { BackendGet } from '@/lib/backend/types'

const WALL_ROUTE = '/mural'

const isPositiveInteger = (value: unknown): value is number =>
  typeof value === 'number' && Number.isInteger(value) && value > 0

const isString = (value: unknown): value is string => typeof value === 'string'

const parsePost = (value: unknown): Post | null => {
  if (typeof value !== 'object' || value === null) return null
  const { author, body, created_at, id, mine, title, updated_at } =
    value as Record<string, unknown>
  if (typeof author !== 'object' || author === null) return null
  const { login, user_id: userId } = author as Record<string, unknown>
  if (
    !isPositiveInteger(id) ||
    !isString(title) ||
    !isString(body) ||
    !isString(created_at) ||
    !isString(updated_at) ||
    typeof mine !== 'boolean' ||
    !isPositiveInteger(userId) ||
    !isString(login)
  ) {
    return null
  }
  return {
    author: { login, userId },
    body,
    createdAt: created_at,
    id,
    mine,
    title,
    updatedAt: updated_at
  }
}

/** The GET /api/posts body as a domain page, or null when it is not one. */
export const parsePostPage = (value: unknown): PostPage | null => {
  if (typeof value !== 'object' || value === null) return null
  const { has_next, items, page, page_size } = value as Record<string, unknown>
  if (
    !Array.isArray(items) ||
    !isPositiveInteger(page) ||
    !isPositiveInteger(page_size) ||
    typeof has_next !== 'boolean'
  ) {
    return null
  }
  const posts = items.map(parsePost)
  if (posts.some(p => p === null)) return null
  return {
    hasNext: has_next,
    items: posts as Post[],
    page,
    pageSize: page_size
  }
}

/**
 * One page of the wall. The backend stays the authority: 401 (the session
 * ended) → login, 403 (the role changed) → forbidden. Anything else that is
 * not a valid 200 throws (fail closed: the page shows an error, never a
 * partial wall).
 */
export const loadPosts = async (
  page: number,
  cookie: string,
  get: BackendGet
): Promise<PostsResult> => {
  if (!isSessionValue(cookie)) {
    return { kind: 'login', url: loginUrl(WALL_ROUTE) }
  }
  const answer = await get(`/api/posts?page=${page}`, cookie)
  if (answer.status === 401) {
    return { kind: 'login', url: loginUrl(WALL_ROUTE, { expired: true }) }
  }
  if (answer.status === 403) return { kind: 'forbidden' }
  const parsed = answer.status === 200 ? parsePostPage(answer.body) : null
  if (!parsed) {
    throw new Error(`unexpected GET /api/posts answer: ${answer.status}`)
  }
  return { kind: 'ok', page: parsed }
}
