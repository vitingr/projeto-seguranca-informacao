import { ShieldCheck } from 'lucide-react'

import { Forbidden } from '@/components/forbidden'
import { AppHeader } from '@/containers/app-header'
import { PostFeed } from '@/containers/post-feed'
import { PostForm } from '@/containers/post-form'
import { requirePosts } from '@/services/posts'
import { requireRole } from '@/services/session'

import type { PostWallProps } from './types'

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
  const footer =
    user.role === 'admin' ? (
      <p
        className="bg-card text-muted-foreground flex items-start gap-2.5 rounded-2xl border px-4 py-3 text-sm text-pretty"
        data-testid="moderation-hint"
      >
        <ShieldCheck
          aria-hidden="true"
          className="text-brand-ink size-4 shrink-0 translate-y-0.5"
        />
        <span>
          Como administrador, você vê todos os posts, inclusive os removidos, e
          pode remover qualquer post informando o motivo. O autor vê o motivo no
          lugar do post, e a remoção vai para a auditoria.
        </span>
      </p>
    ) : (
      <PostForm mode="create" />
    )

  // The wall fills the screen under the header: only the list scrolls, so
  // the composer stays in reach, like a conversation.
  return (
    <div className="flex h-dvh flex-col">
      <AppHeader role={user.role} />
      <main className="mx-auto flex min-h-0 w-full max-w-7xl flex-1 flex-col px-4 py-4 sm:px-6 sm:py-6 lg:px-8">
        <section
          aria-label="Mural"
          className="sm:bg-muted/40 mx-auto flex min-h-0 w-full max-w-4xl flex-1 flex-col gap-3 sm:rounded-3xl sm:border sm:p-5"
        >
          <PostFeed
            footer={footer}
            hasNext={wall.hasNext}
            page={wall.page}
            posts={wall.items}
            viewerRole={user.role}
          />
        </section>
      </main>
    </div>
  )
}
