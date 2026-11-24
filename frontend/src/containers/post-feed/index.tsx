'use client'

import { cn } from 'cn'
import { ArrowDown, ArrowUp, MessagesSquare } from 'lucide-react'
import Link from 'next/link'
import { useState } from 'react'

import { AuthorGlyph } from '@/components/author-glyph'
import { staggerAt } from '@/components/stagger'
import { Button, buttonVariants } from '@/components/ui/shadcn/button'
import { PostItem } from '@/containers/post-item'
import type { Post } from '@/domain/post/types'
import { dayKey, formatDay } from '@/utils/helpers/format-date-time'

import type { DayGroup, FeedAuthor, PostFeedProps } from './types'

const pageHref = (page: number) =>
  page > 1 ? `/mural?pagina=${page}` : '/mural'

/** The authors of the page, the viewer first as "Você", then by order. */
const authorsOf = (posts: Post[]): FeedAuthor[] => {
  const seen = new Map<string, FeedAuthor>()
  for (const post of posts) {
    const { login } = post.author
    if (!seen.has(login)) {
      seen.set(login, { label: post.mine ? 'Você' : login, login })
    }
  }
  return [...seen.values()].sort(
    (a, b) => Number(b.label === 'Você') - Number(a.label === 'Você')
  )
}

/** Oldest first, split by Brasília day, like a conversation. */
const groupByDay = (newestFirst: Post[]): DayGroup[] => {
  const groups: DayGroup[] = []
  for (const post of [...newestFirst].reverse()) {
    const key = dayKey(post.createdAt)
    const last = groups.at(-1)
    if (last?.key === key) {
      last.posts.push(post)
    } else {
      groups.push({ key, label: formatDay(post.createdAt), posts: [post] })
    }
  }
  return groups
}

/**
 * The wall as a conversation: the newest post sits at the bottom, next to
 * the composer, and the list opens scrolled there (flex-col-reverse, no
 * script). The author filter only narrows the page already loaded.
 */
export const PostFeed: React.FC<PostFeedProps> = ({
  footer,
  hasNext,
  page,
  posts,
  viewerRole
}) => {
  const [author, setAuthor] = useState<string | null>(null)
  const authors = authorsOf(posts)
  const shown = author
    ? posts.filter(post => post.author.login === author)
    : posts
  const groups = groupByDay(shown)
  const isAdmin = viewerRole === 'admin'
  // 0 for the newest post: it enters first, from the bottom up.
  const recency = new Map(shown.map((post, index) => [post.id, index]))

  return (
    <>
      <div className="flex flex-col gap-3">
        <div className="flex items-baseline justify-between gap-4">
          <h1 className="text-xl font-semibold tracking-tight">Mural</h1>
          <span className="text-muted-foreground text-xs tabular-nums">
            Página {page}
          </span>
        </div>
        {authors.length > 1 ? (
          <div
            aria-label="Filtrar por autor"
            className="flex gap-1 overflow-x-auto pb-1"
            role="group"
          >
            {[{ label: 'Todos', login: null }, ...authors].map(item => (
              <Button
                aria-pressed={author === item.login}
                className="text-muted-foreground aria-pressed:bg-muted aria-pressed:text-foreground font-normal"
                key={item.login ?? '*'}
                onClick={() => setAuthor(item.login)}
                size="sm"
                variant="ghost"
              >
                {item.login ? <AuthorGlyph login={item.login} /> : null}
                {item.label}
              </Button>
            ))}
          </div>
        ) : null}
      </div>

      <div className="flex min-h-0 flex-1 flex-col-reverse overflow-y-auto overscroll-contain">
        <div
          className="flex shrink-0 grow flex-col justify-end gap-6 py-3"
          data-feed
        >
          {hasNext ? (
            <Link
              className={cn(
                buttonVariants({ size: 'sm', variant: 'outline' }),
                'self-center rounded-full'
              )}
              href={pageHref(page + 1)}
            >
              <ArrowUp aria-hidden="true" data-icon="inline-start" />
              Anteriores
            </Link>
          ) : null}

          {posts.length === 0 ? (
            <div className="motion-safe:animate-rise flex flex-1 flex-col items-center justify-center gap-3 px-6 py-12 text-center">
              <span
                aria-hidden="true"
                className="bg-muted text-muted-foreground flex size-11 items-center justify-center rounded-xl"
              >
                <MessagesSquare className="size-5" strokeWidth={1.75} />
              </span>
              <p className="font-medium" data-testid="wall-empty">
                {page > 1
                  ? 'Não há posts nesta página.'
                  : isAdmin
                    ? 'Ainda não há posts.'
                    : 'Ainda não há posts. Publique o primeiro!'}
              </p>
              <p className="text-muted-foreground max-w-sm text-sm text-pretty">
                {page > 1
                  ? 'Volte para as páginas mais recentes.'
                  : isAdmin
                    ? 'Os posts dos usuários comuns aparecem aqui, com a opção de remover.'
                    : 'Escreva no campo abaixo para começar a conversa.'}
              </p>
            </div>
          ) : (
            groups.map(group => (
              <section
                aria-labelledby={`day-${group.key}`}
                className="flex flex-col gap-3"
                key={group.key}
              >
                <h2
                  className="text-muted-foreground pl-7 text-xs first-letter:uppercase"
                  id={`day-${group.key}`}
                >
                  {group.label}
                </h2>
                <ol className="flex flex-col gap-3">
                  {group.posts.map(post => (
                    <li
                      className={cn(
                        'motion-safe:animate-rise flex gap-3',
                        staggerAt(recency.get(post.id) ?? 0),
                        post.mine && 'sm:justify-end'
                      )}
                      key={post.id}
                    >
                      {post.mine ? null : (
                        <AuthorGlyph
                          className={cn(
                            'translate-y-3.5',
                            post.removal
                              ? 'text-muted-foreground'
                              : 'text-foreground/70'
                          )}
                          login={post.author.login}
                        />
                      )}
                      <PostItem post={post} viewerRole={viewerRole} />
                    </li>
                  ))}
                </ol>
              </section>
            ))
          )}

          {page > 1 ? (
            <Link
              className={cn(
                buttonVariants({ size: 'sm', variant: 'outline' }),
                'self-center rounded-full'
              )}
              href={pageHref(page - 1)}
            >
              Mais recentes
              <ArrowDown aria-hidden="true" data-icon="inline-end" />
            </Link>
          ) : null}
        </div>
      </div>

      {footer}
    </>
  )
}
