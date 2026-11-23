export interface PostAuthor {
  login: string
  userId: number
}

/** The moderation of a post (006 D-2). */
export interface PostRemoval {
  reason: string
  /** ISO 8601 timestamp. */
  removedAt: string
}

export interface Post {
  author: PostAuthor
  body: string
  /** ISO 8601 timestamp. */
  createdAt: string
  id: number
  /** True when the current user wrote the post. */
  mine: boolean
  /** Set when an admin removed the post: it is frozen. */
  removal: PostRemoval | null
  title: string
  /** ISO 8601 timestamp. */
  updatedAt: string
}

export interface PostPage {
  hasNext: boolean
  items: Post[]
  page: number
  pageSize: number
}

export interface PostInput {
  body: string
  title: string
}

export type PostsResult =
  | { kind: 'forbidden' }
  | { kind: 'login'; url: string }
  | { kind: 'ok'; page: PostPage }

/** What a viewer may do on a post: see postActionsFor. */
export type PostActions = 'edit' | 'none' | 'remove'
