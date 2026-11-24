import { cn } from 'cn'
import { ChevronRight, ShieldAlert } from 'lucide-react'

import { Badge } from '@/components/ui/shadcn/badge'
import { formatDateTime, formatTime } from '@/utils/helpers/format-date-time'

import type { PostCardProps } from './types'

// The title, body and removal reason are user content: they are rendered as
// React text only, never as HTML or markdown (005 D-6, S-3).
// whitespace-pre-wrap keeps the line breaks that were typed.
export const PostCard: React.FC<PostCardProps> = ({
  actions,
  className,
  post,
  showRemovedBody = false
}) => {
  const edited = post.updatedAt !== post.createdAt
  const removed = post.removal !== null
  return (
    <article
      className={cn(
        'flex w-full flex-col gap-2.5 rounded-2xl border px-4 py-3 sm:w-fit sm:max-w-xl sm:min-w-80',
        'feed-dim:opacity-45 motion-safe:transition-[opacity,border-color] motion-safe:duration-300',
        post.mine ? 'bg-muted' : 'bg-card',
        removed ? 'border-dashed' : 'hover:border-foreground/20',
        className
      )}
      data-mine={post.mine ? 'true' : undefined}
      data-removed={removed ? 'true' : undefined}
      data-testid="post"
    >
      <header className="flex items-baseline justify-between gap-4">
        <p className="flex min-w-0 items-center gap-2">
          <span className="truncate text-sm font-medium">
            {post.author.login}
          </span>
          {removed ? <Badge variant="warning">Removido</Badge> : null}
        </p>
        <p className="text-muted-foreground shrink-0 text-xs tabular-nums">
          <time
            dateTime={post.createdAt}
            title={formatDateTime(post.createdAt)}
          >
            {formatTime(post.createdAt)}
          </time>
          {edited ? ' · editado' : null}
        </p>
      </header>

      <h3
        className={cn(
          'rounded-lg px-3 py-2 text-sm font-medium break-words',
          post.mine ? 'bg-card' : 'bg-muted',
          removed && 'text-muted-foreground'
        )}
      >
        {post.title}
      </h3>

      {post.removal ? (
        // A removed post shows the notice instead of its body (006 D-2).
        // Admins can still open the body, collapsed by default, to review
        // the moderation; authors never see it again.
        <>
          <p
            className="bg-warning-soft text-warning flex gap-2 rounded-lg px-3 py-2 text-sm break-words whitespace-pre-wrap"
            data-testid="post-removal"
          >
            <ShieldAlert
              aria-hidden="true"
              className="size-4 shrink-0 translate-y-0.5"
            />
            <span>
              <span className="font-medium">Removido pela moderação:</span>{' '}
              {post.removal.reason}
            </span>
          </p>
          {showRemovedBody ? (
            <details data-testid="post-removed-content">
              <summary className="text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 flex w-fit cursor-pointer list-none items-center gap-1 rounded-md text-xs font-medium outline-none focus-visible:ring-3 motion-safe:transition-colors">
                <ChevronRight
                  aria-hidden="true"
                  className="size-3.5 in-open:rotate-90 motion-safe:transition-transform"
                />
                Ver conteúdo removido
              </summary>
              <div className="pt-2">
                <p
                  className="text-foreground rounded-lg border border-dashed px-3 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap"
                  data-testid="post-removed-body"
                >
                  {post.body}
                </p>
              </div>
            </details>
          ) : null}
        </>
      ) : (
        <p
          className="text-foreground/90 text-sm leading-relaxed break-words whitespace-pre-wrap"
          data-testid="post-body"
        >
          {post.body}
        </p>
      )}
      {actions ? (
        <div className="flex flex-wrap items-center gap-1.5">{actions}</div>
      ) : null}
    </article>
  )
}
