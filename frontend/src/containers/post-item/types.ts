import type { Post } from '@/domain/post/types'
import type { Role } from '@/domain/roles/types'

export interface PostItemProps {
  className?: string
  post: Post
  /** The role of the current user: it picks the actions (006 D-1). */
  viewerRole: Role
}
