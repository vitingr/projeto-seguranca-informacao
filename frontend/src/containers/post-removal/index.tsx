'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { useForm } from 'react-hook-form'

import { Button } from '@/components/ui/shadcn/button'
import { FormMessage } from '@/components/ui/shadcn/field'
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
      className="border-border flex w-full flex-col gap-3 rounded-lg border p-3"
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <div className="flex flex-col gap-1.5">
        <label className="text-sm font-medium" htmlFor={id}>
          Motivo
        </label>
        <textarea
          aria-describedby={reasonError ? `${id}-error` : undefined}
          aria-invalid={reasonError ? true : undefined}
          className="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive min-h-20 rounded-lg border px-3 py-2 text-sm outline-none focus-visible:ring-3"
          id={id}
          rows={3}
          {...register('reason')}
        />
        {reasonError ? (
          <p className="text-destructive text-sm" id={`${id}-error`}>
            {reasonError}
          </p>
        ) : null}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button disabled={isSubmitting} type="submit" variant="destructive">
          {isSubmitting ? 'Removendo…' : 'Confirmar remoção'}
        </Button>
        <Button
          disabled={isSubmitting}
          onClick={onCancel}
          type="button"
          variant="outline"
        >
          Cancelar
        </Button>
      </div>
    </form>
  )
}
