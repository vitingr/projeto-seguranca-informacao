import type { ReactNode } from 'react'

import type { Post } from '@/domain/post/types'

export interface PostCardProps {
  /** Buttons shown under the post (the author's edit and delete). */
  actions?: ReactNode
  className?: string
  post: Post
  /**
   * Lets the viewer open the body of a removed post, collapsed (admins
   * only). Ignored for a post that is not removed.
   */
  showRemovedBody?: boolean
}
