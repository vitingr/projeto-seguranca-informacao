import Link from 'next/link'

import { Forbidden } from '@/components/forbidden'
import { buttonVariants } from '@/components/ui/shadcn/button'
import { AppHeader } from '@/containers/app-header'
import { PostForm } from '@/containers/post-form'
import { PostItem } from '@/containers/post-item'
import { requirePosts } from '@/services/posts'
import { requireRole } from '@/services/session'

import type { PostWallProps } from './types'

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
      <main className="mx-auto w-full max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-8">
          <div className="flex flex-col gap-2">
            <h1 className="text-2xl font-semibold">Mural</h1>
            {isAdmin ? (
              <p
                className="text-muted-foreground"
                data-testid="moderation-hint"
              >
                Como administrador, você vê todos os posts, inclusive os
                removidos, e pode remover qualquer post informando o motivo.
              </p>
            ) : null}
          </div>
          {isAdmin ? null : (
            <section aria-labelledby="new-post" className="flex flex-col gap-4">
              <h2 className="text-lg font-semibold" id="new-post">
                Novo post
              </h2>
              <PostForm mode="create" />
            </section>
          )}
          <section aria-labelledby="posts" className="flex flex-col gap-4">
            <h2 className="text-lg font-semibold" id="posts">
              Posts
            </h2>
            {wall.items.length === 0 ? (
              <p className="text-muted-foreground" data-testid="wall-empty">
                {wall.page > 1
                  ? 'Não há posts nesta página.'
                  : isAdmin
                    ? 'Ainda não há posts.'
                    : 'Ainda não há posts. Publique o primeiro!'}
              </p>
            ) : (
              <ul className="flex flex-col gap-4">
                {wall.items.map(post => (
                  <li key={post.id}>
                    <PostItem post={post} viewerRole={user.role} />
                  </li>
                ))}
              </ul>
            )}
          </section>
          {wall.page > 1 || wall.hasNext ? (
            <nav aria-label="Paginação" className="flex flex-wrap gap-2">
              {wall.page > 1 ? (
                <Link
                  className={buttonVariants({ variant: 'outline' })}
                  href={pageHref(wall.page - 1)}
                >
                  Mais recentes
                </Link>
              ) : null}
              {wall.hasNext ? (
                <Link
                  className={buttonVariants({ variant: 'outline' })}
                  href={pageHref(wall.page + 1)}
                >
                  Anteriores
                </Link>
              ) : null}
            </nav>
          ) : null}
        </div>
      </main>
    </>
  )
}
