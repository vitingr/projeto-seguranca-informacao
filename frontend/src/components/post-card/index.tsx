import { formatDateTime } from '@/utils/helpers/format-date-time'

import type { PostCardProps } from './types'

// The title, body and removal reason are user content: they are rendered as
// React text only, never as HTML or markdown (005 D-6, S-3).
// whitespace-pre-wrap keeps the line breaks that were typed.
export const PostCard: React.FC<PostCardProps> = ({
  actions,
  post,
  showRemovedBody = false
}) => {
  const edited = post.updatedAt !== post.createdAt
  const removed = post.removal !== null
  return (
    <article
      className={
        removed
          ? 'border-border bg-muted text-muted-foreground flex flex-col gap-3 rounded-lg border border-dashed p-4'
          : 'border-border flex flex-col gap-3 rounded-lg border p-4'
      }
      data-removed={removed ? 'true' : undefined}
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
      {post.removal ? (
        // A removed post shows the notice instead of its body (006 D-2).
        // Admins can still open the body, collapsed by default, to review
        // the moderation; authors never see it again.
        <>
          <p
            className="text-foreground break-words whitespace-pre-wrap"
            data-testid="post-removal"
          >
            <span className="font-medium">Removido pela moderação:</span>{' '}
            {post.removal.reason}
          </p>
          {showRemovedBody ? (
            <details data-testid="post-removed-content">
              <summary className="text-foreground focus-visible:ring-ring/50 w-fit cursor-pointer rounded-sm text-sm font-medium outline-none focus-visible:ring-3">
                Ver conteúdo removido
              </summary>
              <div className="pt-2">
                <p
                  className="border-border bg-background text-foreground rounded-md border p-3 break-words whitespace-pre-wrap"
                  data-testid="post-removed-body"
                >
                  {post.body}
                </p>
              </div>
            </details>
          ) : null}
        </>
      ) : (
        <p className="break-words whitespace-pre-wrap" data-testid="post-body">
          {post.body}
        </p>
      )}
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </article>
  )
}
