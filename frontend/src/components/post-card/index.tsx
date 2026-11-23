import { formatDateTime } from '@/utils/helpers/format-date-time'

import type { PostCardProps } from './types'

// The title and body are user content: they are rendered as React text
// only, never as HTML or markdown (005 D-6, S-3). whitespace-pre-wrap keeps
// the line breaks the author typed.
export const PostCard: React.FC<PostCardProps> = ({ actions, post }) => {
  const edited = post.updatedAt !== post.createdAt
  return (
    <article
      className="border-border flex flex-col gap-3 rounded-lg border p-4"
      data-testid="post"
    >
      <header className="flex flex-col gap-1">
        <h2 className="text-lg font-semibold break-words">{post.title}</h2>
        <p className="text-muted-foreground text-sm">
          {post.author.login} ·{' '}
          <time dateTime={post.createdAt}>
            {formatDateTime(post.createdAt)}
          </time>
          {edited ? ' · editado' : null}
        </p>
      </header>
      <p className="break-words whitespace-pre-wrap" data-testid="post-body">
        {post.body}
      </p>
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </article>
  )
}
