import type { Role } from '@/domain/roles/types'

import type { Post, PostActions } from './types'

// The backend's post rules (005 D-4, D-5, 006 D-2), checked before sending
// so the user gets a field message. The backend stays the authority.

export const POST_TITLE_MAX = 120
export const POST_BODY_MAX = 2000
export const POST_PAGE_MAX = 10000
/** Longest moderation reason, in code points after trimming (006 D-2). */
export const REMOVAL_REASON_MAX = 500

/** Length as the backend counts it: code points after trimming. */
export const postLength = (value: string) => [...value.trim()].length

// Unicode control characters (Cc), like Go's unicode.IsControl.
const CONTROL = /\p{Cc}/u
const CONTROL_EXCEPT_LINE_BREAKS_AND_TAB = /(?![\n\r\t])\p{Cc}/u

export const titleHasControl = (value: string) => CONTROL.test(value)

export const bodyHasControl = (value: string) =>
  CONTROL_EXCEPT_LINE_BREAKS_AND_TAB.test(value)

/** A removal reason follows the body rule (006 D-2). */
export const reasonHasControl = bodyHasControl

/**
 * A list page in the URL (`?pagina=` of the wall and of the audit log): an
 * integer from 1 to POST_PAGE_MAX. Anything else, or no value, is page 1.
 */
export const parsePageParam = (raw: string | string[] | undefined): number => {
  if (typeof raw !== 'string' || !/^[1-9]\d{0,4}$/.test(raw)) return 1
  const page = Number(raw)
  return page <= POST_PAGE_MAX ? page : 1
}

/**
 * The actions a viewer gets on a post (006 D-1, D-2): a comum author edits
 * and deletes their own post, an admin only removes, and a removed post is
 * frozen for everyone. The backend enforces the same rules again.
 */
export const postActionsFor = (
  post: Pick<Post, 'mine' | 'removal'>,
  role: Role
): PostActions => {
  if (post.removal !== null) return 'none'
  if (role === 'admin') return 'remove'
  return post.mine ? 'edit' : 'none'
}
