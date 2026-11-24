import { cn } from 'cn'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  MessagesSquare,
  ShieldCheck
} from 'lucide-react'
import Link from 'next/link'

import { Eyebrow } from '@/components/eyebrow'
import { Forbidden } from '@/components/forbidden'
import { staggerAt } from '@/components/stagger'
import { buttonVariants } from '@/components/ui/shadcn/button'
import { AppHeader } from '@/containers/app-header'
import { PostForm } from '@/containers/post-form'
import { PostItem } from '@/containers/post-item'
import { POST_BODY_MAX, POST_TITLE_MAX } from '@/domain/post'
import { requirePosts } from '@/services/posts'
import { requireRole } from '@/services/session'

import type { PostWallProps } from './types'

const RISE = 'motion-safe:animate-rise'

const MODERATION_RULES = [
  'O motivo é obrigatório e aparece para o autor no lugar do post.',
  'Um post removido fica congelado: ninguém edita nem exclui.',
  'Cada remoção vai para a auditoria, com você como ator.'
] as const

const pageHref = (page: number) =>
  page > 1 ? `/mural?pagina=${page}` : '/mural'

// Both roles (006 D-1): comum users publish, edit and delete their own
// posts; admins only read the wall, removed posts included, and remove
// posts. The backend enforces the same rules on every call, and answers 403
// if the role changed in between.
export const PostWall: React.FC<PostWallProps> = async ({ page }) => {
  const { allowed, user } = await requireRole('/mural')
  const result = allowed ? await requirePosts(page) : null

  if (result?.kind !== 'ok') {
    return (
      <>
        <AppHeader role={user.role} />
        <Forbidden role={user.role} />
      </>
    )
  }

  const wall = result.page
  const isAdmin = user.role === 'admin'
  return (
    <>
      <AppHeader role={user.role} />
      <main className="mx-auto flex w-full max-w-7xl flex-col gap-10 px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
        <div
          className={cn(
            RISE,
            'flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between'
          )}
        >
          <div className="flex flex-col gap-4">
            <Eyebrow>Comunidade</Eyebrow>
            <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
              Mural
            </h1>
            <p className="text-muted-foreground max-w-xl text-pretty">
              Posts de todos os usuários, do mais recente para o mais antigo.
            </p>
          </div>
          <span className="text-muted-foreground text-sm">
            Página {wall.page}
          </span>
        </div>

        <div className="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:gap-10">
          <aside
            className={cn(
              RISE,
              'motion-safe:anim-delay-75',
              'flex flex-col gap-4 self-start lg:sticky lg:top-24 lg:col-span-4 lg:col-start-9 lg:row-start-1'
            )}
          >
            {isAdmin ? (
              <section
                aria-labelledby="moderation"
                className="bg-card flex flex-col gap-5 rounded-3xl border p-6"
              >
                <div className="flex items-center gap-3">
                  <span
                    aria-hidden="true"
                    className="bg-brand-soft text-brand-ink flex size-10 items-center justify-center rounded-xl"
                  >
                    <ShieldCheck className="size-5" strokeWidth={1.75} />
                  </span>
                  <h2
                    className="text-lg font-semibold tracking-tight"
                    id="moderation"
                  >
                    Moderação
                  </h2>
                </div>
                <p
                  className="text-muted-foreground text-sm text-pretty"
                  data-testid="moderation-hint"
                >
                  Como administrador, você vê todos os posts, inclusive os
                  removidos, e pode remover qualquer post informando o motivo.
                </p>
                <ul className="flex flex-col gap-3 border-t pt-5">
                  {MODERATION_RULES.map(rule => (
                    <li className="flex gap-2.5 text-sm" key={rule}>
                      <Check
                        aria-hidden="true"
                        className="text-brand-ink size-4 shrink-0 translate-y-0.5"
                      />
                      {rule}
                    </li>
                  ))}
                </ul>
              </section>
            ) : (
              <section
                aria-labelledby="new-post"
                className="bg-card flex flex-col gap-5 rounded-3xl border p-6"
              >
                <div className="flex flex-col gap-1.5">
                  <h2
                    className="text-lg font-semibold tracking-tight"
                    id="new-post"
                  >
                    Novo post
                  </h2>
                  <p className="text-muted-foreground text-sm text-pretty">
                    Título de até {POST_TITLE_MAX} caracteres e texto de até{' '}
                    {POST_BODY_MAX}. Depois, só você pode editar ou excluir.
                  </p>
                </div>
                <PostForm mode="create" />
              </section>
            )}
          </aside>

          <section
            aria-labelledby="posts"
            className="flex flex-col gap-4 lg:col-span-8 lg:col-start-1 lg:row-start-1"
          >
            <h2 className="sr-only" id="posts">
              Posts
            </h2>
            {wall.items.length === 0 ? (
              <div
                className={cn(
                  RISE,
                  'flex flex-col items-center gap-4 rounded-3xl border border-dashed px-6 py-16 text-center'
                )}
              >
                <span
                  aria-hidden="true"
                  className="bg-brand-soft text-brand-ink flex size-12 items-center justify-center rounded-2xl"
                >
                  <MessagesSquare className="size-6" strokeWidth={1.75} />
                </span>
                <p
                  className="text-lg font-semibold tracking-tight"
                  data-testid="wall-empty"
                >
                  {wall.page > 1
                    ? 'Não há posts nesta página.'
                    : isAdmin
                      ? 'Ainda não há posts.'
                      : 'Ainda não há posts. Publique o primeiro!'}
                </p>
                <p className="text-muted-foreground max-w-sm text-sm text-pretty">
                  {wall.page > 1
                    ? 'Volte para as páginas mais recentes.'
                    : isAdmin
                      ? 'Os posts dos usuários comuns aparecem aqui, com a opção de remover.'
                      : 'Use o formulário “Novo post” para começar a conversa.'}
                </p>
              </div>
            ) : (
              <ul className="flex flex-col gap-4">
                {wall.items.map((post, index) => (
                  <li key={post.id}>
                    <PostItem
                      className={cn(RISE, staggerAt(index + 1))}
                      post={post}
                      viewerRole={user.role}
                    />
                  </li>
                ))}
              </ul>
            )}

            {wall.page > 1 || wall.hasNext ? (
              <nav
                aria-label="Paginação"
                className="flex items-center justify-between gap-3 border-t pt-6"
              >
                {wall.page > 1 ? (
                  <Link
                    className={buttonVariants({
                      size: 'lg',
                      variant: 'outline'
                    })}
                    href={pageHref(wall.page - 1)}
                  >
                    <ArrowLeft aria-hidden="true" data-icon="inline-start" />
                    Mais recentes
                  </Link>
                ) : (
                  <span />
                )}
                <span className="text-muted-foreground hidden text-sm sm:inline">
                  Página {wall.page}
                </span>
                {wall.hasNext ? (
                  <Link
                    className={buttonVariants({
                      size: 'lg',
                      variant: 'outline'
                    })}
                    href={pageHref(wall.page + 1)}
                  >
                    Anteriores
                    <ArrowRight aria-hidden="true" data-icon="inline-end" />
                  </Link>
                ) : (
                  <span />
                )}
              </nav>
            ) : null}
          </section>
        </div>
      </main>
    </>
  )
}
