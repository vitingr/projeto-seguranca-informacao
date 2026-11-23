'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'

import { Button } from '@/components/ui/shadcn/button'
import { Field, FormMessage } from '@/components/ui/shadcn/field'
import { messageFor } from '@/domain/messages'
import { api } from '@/repositories/api'
import { zodResolver } from '@hookform/resolvers/zod'

import { postSchema } from './schemas'
import type { PostFormProps, PostValues } from './types'

const EMPTY: PostValues = { body: '', title: '' }

/** Creates a post, or edits one of the user's posts in place. */
export const PostForm = (props: PostFormProps) => {
  // Opt out of the React Compiler, like the registration form: with it,
  // react-hook-form's reset() did not clear the inputs.
  'use no memo'

  const router = useRouter()
  const editing = props.mode === 'edit'
  const prefix = editing ? `post-${props.post.id}` : 'new-post'
  const [error, setError] = useState<string | null>(null)
  const [created, setCreated] = useState<number>(0)
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
    reset
  } = useForm<PostValues>({
    defaultValues: editing
      ? { body: props.post.body, title: props.post.title }
      : EMPTY,
    resolver: zodResolver(postSchema)
  })

  // Reset after the submission has settled (see the registration form).
  useEffect(() => {
    if (created > 0) reset(EMPTY)
  }, [created, reset])

  const onSubmit = async (values: PostValues) => {
    setError(null)
    const result = editing
      ? await api.updatePost(props.post.id, values)
      : await api.createPost(values)
    if (!result.ok) {
      setError(messageFor(result.error, 'post'))
      return
    }
    if (editing) {
      props.onSaved()
    } else {
      setCreated(n => n + 1)
    }
    router.refresh()
  }

  const bodyError = errors.body?.message
  return (
    <form
      aria-label={editing ? 'Editar post' : 'Novo post'}
      className="flex flex-col gap-4"
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {error ? <FormMessage tone="error">{error}</FormMessage> : null}
      <Field
        autoComplete="off"
        error={errors.title?.message}
        id={`${prefix}-title`}
        label="Título"
        {...register('title')}
      />
      <div className="flex flex-col gap-1.5">
        <label className="text-sm font-medium" htmlFor={`${prefix}-body`}>
          Texto
        </label>
        <textarea
          aria-describedby={bodyError ? `${prefix}-body-error` : undefined}
          aria-invalid={bodyError ? true : undefined}
          className="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive min-h-24 rounded-lg border px-3 py-2 text-sm outline-none focus-visible:ring-3"
          id={`${prefix}-body`}
          rows={4}
          {...register('body')}
        />
        {bodyError ? (
          <p className="text-destructive text-sm" id={`${prefix}-body-error`}>
            {bodyError}
          </p>
        ) : null}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button disabled={isSubmitting} size="lg" type="submit">
          {editing
            ? isSubmitting
              ? 'Salvando…'
              : 'Salvar'
            : isSubmitting
              ? 'Publicando…'
              : 'Publicar'}
        </Button>
        {editing ? (
          <Button
            disabled={isSubmitting}
            onClick={props.onCancel}
            size="lg"
            type="button"
            variant="outline"
          >
            Cancelar
          </Button>
        ) : null}
      </div>
    </form>
  )
}
