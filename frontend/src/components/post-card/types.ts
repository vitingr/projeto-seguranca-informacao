import type { ReactNode } from 'react'

import type { Post } from '@/domain/post/types'

export interface PostCardProps {
  /** Buttons shown under the post (the author's edit and delete). */
  actions?: ReactNode
  post: Post
}
