import type { ReactNode } from 'react'

import type { Post } from '@/domain/post/types'
import type { Role } from '@/domain/roles/types'

export interface PostFeedProps {
  /** What sits under the list: the composer, or the moderation note. */
  footer: ReactNode
  hasNext: boolean
  page: number
  /** The page of the wall, newest first, as the API answers. */
  posts: Post[]
  viewerRole: Role
}

export interface FeedAuthor {
  label: string
  login: string
}

export interface DayGroup {
  /** "aaaa-mm-dd" of the Brasília day. */
  key: string
  label: string
  posts: Post[]
}
