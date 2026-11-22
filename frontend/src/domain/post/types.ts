export interface PostAuthor {
  login: string
  userId: number
}

export interface Post {
  author: PostAuthor
  body: string
  /** ISO 8601 timestamp. */
  createdAt: string
  id: number
  /** True when the current user wrote the post. */
  mine: boolean
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
