'use client'

import { CircleAlert } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { Button } from '@/components/ui/shadcn/button'
import { FormMessage } from '@/components/ui/shadcn/field'
import { Textarea } from '@/components/ui/shadcn/textarea'
import { messageFor } from '@/domain/messages'
import { api } from '@/repositories/api'
import { zodResolver } from '@hookform/resolvers/zod'

import { removalSchema } from './schemas'
import type { PostRemovalProps, RemovalValues } from './types'

/**
 * The admin's inline removal of a post (006 D-2): a reason, then "Confirmar
 * remoção" or "Cancelar", never window.confirm (005 D-7). The backend checks
 * the role again and answers 403 to anyone else.
 */
export const PostRemoval: React.FC<PostRemovalProps> = ({
  onCancel,
  postId
}) => {
  // Opt out of the React Compiler, like the other react-hook-form forms.
  'use no memo'

  const router = useRouter()
  const [error, setError] = useState<string | null>(null)
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register
  } = useForm<RemovalValues>({
    defaultValues: { reason: '' },
    resolver: zodResolver(removalSchema)
  })

  const onSubmit = async ({ reason }: RemovalValues) => {
    setError(null)
    const result = await api.removePost(postId, reason)
    if (!result.ok) {
      setError(messageFor(result.error, 'removal'))
      return
    }
    router.refresh()
  }

  const id = `removal-${postId}-reason`
  const reasonError = errors.reason?.message
  return (
    <form
      aria-label="Remover post"
      className="bg-background motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-top-1 flex w-full flex-col gap-3 rounded-xl border p-3"
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <div className="flex flex-col gap-2">
        <label className="text-xs font-medium" htmlFor={id}>
          Motivo
        </label>
        <Textarea
          aria-describedby={reasonError ? `${id}-error` : `${id}-hint`}
          aria-invalid={reasonError ? true : undefined}
          className="min-h-20 text-sm"
          id={id}
          rows={3}
          {...register('reason')}
        />
        {reasonError ? (
          <p
            className="text-destructive motion-safe:animate-in motion-safe:fade-in flex items-start gap-1.5 text-xs"
            id={`${id}-error`}
          >
            <CircleAlert aria-hidden="true" className="size-3.5 shrink-0" />
            {reasonError}
          </p>
        ) : (
          <p
            className="text-muted-foreground text-xs text-pretty"
            id={`${id}-hint`}
          >
            O autor verá este motivo no lugar do post. A remoção vai para a
            auditoria.
          </p>
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          disabled={isSubmitting}
          size="sm"
          type="submit"
          variant="danger"
        >
          {isSubmitting ? 'Removendo…' : 'Confirmar remoção'}
        </Button>
        <Button
          disabled={isSubmitting}
          onClick={onCancel}
          size="sm"
          type="button"
          variant="ghost"
        >
          Cancelar
        </Button>
      </div>
    </form>
  )
}
