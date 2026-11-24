'use client'

import { cn } from 'cn'
import { Pencil, ShieldX, Trash2 } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { PostCard } from '@/components/post-card'
import { Button } from '@/components/ui/shadcn/button'
import { FormMessage } from '@/components/ui/shadcn/field'
import { PostForm } from '@/containers/post-form'
import { PostRemoval } from '@/containers/post-removal'
import { messageFor } from '@/domain/messages'
import { postActionsFor } from '@/domain/post'
import { api } from '@/repositories/api'

import type { PostItemProps } from './types'

/**
 * One post of the wall, with the actions of the viewer's role:
 * - comum: "Editar" and "Excluir" on their own posts (`mine`); the backend
 *   checks the owner again on every change (005 D-3). Deleting asks for a
 *   second click, never window.confirm (D-7).
 * - admin: only "Remover", with a reason (006 D-1, D-2).
 * A removed post is frozen: it has no actions for anyone. Admins can open
 * its body, collapsed; its author only sees the notice with the reason.
 */
export const PostItem: React.FC<PostItemProps> = ({
  className,
  post,
  viewerRole
}) => {
  const router = useRouter()
  const [editing, setEditing] = useState<boolean>(false)
  const [removing, setRemoving] = useState<boolean>(false)
  const [confirming, setConfirming] = useState<boolean>(false)
  const [pending, setPending] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)

  const allowed = postActionsFor(post, viewerRole)
  const canEdit = allowed === 'edit'
  const canRemove = allowed === 'remove'

  if (editing && canEdit) {
    return (
      <div
        className="flex w-full flex-col gap-2 sm:max-w-xl"
        data-testid="post-edit"
      >
        <span className="text-muted-foreground flex items-center gap-1.5 px-1 text-xs">
          <Pencil aria-hidden="true" className="size-3" />
          Editando
        </span>
        <PostForm
          mode="edit"
          onCancel={() => setEditing(false)}
          onSaved={() => setEditing(false)}
          post={post}
        />
      </div>
    )
  }

  const remove = async () => {
    setPending(true)
    setError(null)
    const result = await api.deletePost(post.id)
    setPending(false)
    if (!result.ok) {
      setError(messageFor(result.error, 'post'))
      setConfirming(false)
      return
    }
    router.refresh()
  }

  let actions: React.ReactNode = null
  if (canRemove) {
    actions = removing ? (
      <PostRemoval onCancel={() => setRemoving(false)} postId={post.id} />
    ) : (
      <Button onClick={() => setRemoving(true)} size="xs" variant="destructive">
        <ShieldX aria-hidden="true" data-icon="inline-start" />
        Remover
      </Button>
    )
  } else if (canEdit) {
    actions = confirming ? (
      <div className="motion-safe:animate-in motion-safe:fade-in flex flex-wrap items-center gap-1.5">
        <span className="text-destructive px-1 text-xs font-medium">
          Excluir este post?
        </span>
        <Button disabled={pending} onClick={remove} size="xs" variant="danger">
          {pending ? 'Excluindo…' : 'Confirmar exclusão'}
        </Button>
        <Button
          disabled={pending}
          onClick={() => setConfirming(false)}
          size="xs"
          variant="ghost"
        >
          Cancelar
        </Button>
      </div>
    ) : (
      <>
        <Button onClick={() => setEditing(true)} size="xs" variant="outline">
          <Pencil aria-hidden="true" data-icon="inline-start" />
          Editar
        </Button>
        <Button onClick={() => setConfirming(true)} size="xs" variant="outline">
          <Trash2 aria-hidden="true" data-icon="inline-start" />
          Excluir
        </Button>
      </>
    )
  }

  return (
    <div
      className={cn(
        'flex w-full flex-col gap-2 sm:w-auto',
        post.mine && 'sm:items-end'
      )}
    >
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <PostCard
        actions={actions}
        className={className}
        post={post}
        showRemovedBody={viewerRole === 'admin'}
      />
    </div>
  )
}
