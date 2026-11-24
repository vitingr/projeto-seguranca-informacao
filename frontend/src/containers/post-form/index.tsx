'use client'

import { cn } from 'cn'
import { CircleAlert, Send } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'

import { Button } from '@/components/ui/shadcn/button'
import { Field, FormMessage } from '@/components/ui/shadcn/field'
import { Textarea } from '@/components/ui/shadcn/textarea'
import { messageFor } from '@/domain/messages'
import { POST_BODY_MAX, postLength } from '@/domain/post'
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
    control,
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
  // Display only: the schema above still decides what is valid.
  const bodyLength = postLength(useWatch({ control, name: 'body' }) ?? '')
  return (
    <form
      aria-label={editing ? 'Editar post' : 'Novo post'}
      className="flex flex-col gap-5"
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
      <div className="flex flex-col gap-2">
        <label className="text-sm font-medium" htmlFor={`${prefix}-body`}>
          Texto
        </label>
        <Textarea
          aria-describedby={
            bodyError ? `${prefix}-body-error` : `${prefix}-body-count`
          }
          aria-invalid={bodyError ? true : undefined}
          className="min-h-32"
          id={`${prefix}-body`}
          rows={4}
          {...register('body')}
        />
        <div className="flex items-start justify-between gap-3">
          {bodyError ? (
            <p
              className="text-destructive motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-top-1 flex items-start gap-1.5 text-sm"
              id={`${prefix}-body-error`}
            >
              <CircleAlert
                aria-hidden="true"
                className="size-4 shrink-0 translate-y-0.5"
              />
              {bodyError}
            </p>
          ) : (
            <span />
          )}
          <span
            className={cn(
              'text-xs tabular-nums',
              bodyLength > POST_BODY_MAX
                ? 'text-destructive'
                : 'text-muted-foreground'
            )}
            id={`${prefix}-body-count`}
          >
            {bodyLength}/{POST_BODY_MAX}
          </span>
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          className={editing ? undefined : 'w-full sm:w-auto'}
          disabled={isSubmitting}
          size="xl"
          type="submit"
        >
          {editing ? null : (
            <Send aria-hidden="true" data-icon="inline-start" />
          )}
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
            size="xl"
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
