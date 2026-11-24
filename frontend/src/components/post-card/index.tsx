import { cn } from 'cn'
import { ChevronRight, ShieldAlert } from 'lucide-react'

import { Badge } from '@/components/ui/shadcn/badge'
import { formatDateTime } from '@/utils/helpers/format-date-time'

import type { PostCardProps } from './types'

const initials = (login: string) =>
  login
    .split(/[._-]/)
    .filter(Boolean)
    .slice(0, 2)
    .map(part => part[0])
    .join('')
    .toUpperCase() || login.slice(0, 2).toUpperCase()

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
        'flex flex-col gap-4 rounded-2xl border p-5 motion-safe:transition-colors motion-safe:duration-300 sm:p-6',
        removed
          ? 'bg-muted border-dashed'
          : 'bg-card hover:border-foreground/20',
        className
      )}
      data-removed={removed ? 'true' : undefined}
      data-testid="post"
    >
      <div className="flex items-center gap-3">
        <span
          className={cn(
            'flex size-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold',
            removed
              ? 'bg-background text-muted-foreground border'
              : 'bg-brand-soft text-brand-ink'
          )}
          aria-hidden="true"
        >
          {initials(post.author.login)}
        </span>
        <p className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm font-medium">
            {post.author.login}
          </span>
          <span className="text-muted-foreground text-xs">
            <time dateTime={post.createdAt}>
              {formatDateTime(post.createdAt)}
            </time>
            {edited ? ' · editado' : null}
          </span>
        </p>
        {removed ? (
          <Badge className="h-6 px-2.5" variant="warning">
            Removido
          </Badge>
        ) : null}
      </div>

      <h2
        className={cn(
          'text-lg font-semibold tracking-tight break-words',
          removed && 'text-muted-foreground'
        )}
      >
        {post.title}
      </h2>

      {post.removal ? (
        // A removed post shows the notice instead of its body (006 D-2).
        // Admins can still open the body, collapsed by default, to review
        // the moderation; authors never see it again.
        <>
          <p
            className="bg-warning-soft text-warning flex gap-2.5 rounded-xl px-4 py-3 text-sm break-words whitespace-pre-wrap"
            data-testid="post-removal"
          >
            <ShieldAlert
              aria-hidden="true"
              className="size-4 shrink-0 translate-y-0.5"
            />
            <span>
              <span className="font-semibold">Removido pela moderação:</span>{' '}
              {post.removal.reason}
            </span>
          </p>
          {showRemovedBody ? (
            <details data-testid="post-removed-content">
              <summary className="text-foreground focus-visible:ring-ring/50 flex w-fit cursor-pointer list-none items-center gap-1.5 rounded-md text-sm font-medium outline-none focus-visible:ring-3">
                <ChevronRight
                  aria-hidden="true"
                  className="size-4 in-open:rotate-90 motion-safe:transition-transform"
                />
                Ver conteúdo removido
              </summary>
              <div className="pt-3">
                <p
                  className="bg-card text-foreground rounded-xl border p-4 text-sm leading-relaxed break-words whitespace-pre-wrap"
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
          className="text-foreground/85 leading-relaxed break-words whitespace-pre-wrap"
          data-testid="post-body"
        >
          {post.body}
        </p>
      )}
      {actions ? (
        <div className="flex flex-wrap items-center gap-2 border-t pt-4">
          {actions}
        </div>
      ) : null}
    </article>
  )
}
