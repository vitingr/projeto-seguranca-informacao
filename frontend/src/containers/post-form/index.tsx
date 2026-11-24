'use client'

import { cn } from 'cn'
import { ArrowUp, CircleAlert, Users } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'

import { Button } from '@/components/ui/shadcn/button'
import { FormMessage } from '@/components/ui/shadcn/field'
import { messageFor } from '@/domain/messages'
import { POST_BODY_MAX, POST_TITLE_MAX, postLength } from '@/domain/post'
import { api } from '@/repositories/api'
import { zodResolver } from '@hookform/resolvers/zod'

import { postSchema } from './schemas'
import type { FieldErrorProps, PostFormProps, PostValues } from './types'

const EMPTY: PostValues = { body: '', title: '' }

const FieldError = ({ children, id }: FieldErrorProps) => (
  <p
    className="text-destructive motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-top-1 flex items-start gap-1.5 text-xs"
    id={id}
  >
    <CircleAlert aria-hidden="true" className="size-3.5 shrink-0" />
    {children}
  </p>
)

/**
 * Creates a post (the composer under the wall), or edits one of the user's
 * posts in place, with the same layout inside the post.
 */
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

  // Ctrl/⌘ + Enter sends from the text; Escape leaves an edit.
  const onKeyDown = (event: React.KeyboardEvent<HTMLFormElement>) => {
    if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
      event.preventDefault()
      event.currentTarget.requestSubmit()
    } else if (event.key === 'Escape' && editing && !isSubmitting) {
      props.onCancel()
    }
  }

  const titleError = errors.title?.message
  const bodyError = errors.body?.message
  // Display only: the schema above still decides what is valid.
  const title = useWatch({ control, name: 'title' }) ?? ''
  const bodyLength = postLength(useWatch({ control, name: 'body' }) ?? '')
  const ready = postLength(title) > 0 && bodyLength > 0
  return (
    <form
      aria-label={editing ? 'Editar post' : 'Novo post'}
      className="bg-card focus-within:border-ring/60 focus-within:ring-ring/15 flex w-full flex-col rounded-2xl border focus-within:ring-4 motion-safe:transition-[border-color,box-shadow]"
      onKeyDown={onKeyDown}
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      {error ? (
        <div className="px-3 pt-3">
          <FormMessage tone="error">{error}</FormMessage>
        </div>
      ) : null}
      <label className="sr-only" htmlFor={`${prefix}-title`}>
        Título
      </label>
      <input
        aria-describedby={titleError ? `${prefix}-title-error` : undefined}
        aria-invalid={titleError ? true : undefined}
        autoComplete="off"
        className="placeholder:text-muted-foreground w-full bg-transparent px-4 pt-3.5 pb-1 text-sm font-medium outline-none"
        id={`${prefix}-title`}
        placeholder={`Título, até ${POST_TITLE_MAX} caracteres`}
        {...register('title')}
      />
      <label className="sr-only" htmlFor={`${prefix}-body`}>
        Texto
      </label>
      <textarea
        aria-describedby={
          bodyError ? `${prefix}-body-error` : `${prefix}-body-count`
        }
        aria-invalid={bodyError ? true : undefined}
        className="placeholder:text-muted-foreground field-sizing-content max-h-60 min-h-12 w-full resize-none bg-transparent px-4 py-1.5 text-sm leading-relaxed outline-none"
        id={`${prefix}-body`}
        placeholder="Escreva para o mural…"
        rows={editing ? 3 : 2}
        {...register('body')}
      />
      {titleError || bodyError ? (
        <div className="flex flex-col gap-1 px-4 pt-1">
          {titleError ? (
            <FieldError id={`${prefix}-title-error`}>{titleError}</FieldError>
          ) : null}
          {bodyError ? (
            <FieldError id={`${prefix}-body-error`}>{bodyError}</FieldError>
          ) : null}
        </div>
      ) : null}
      <div className="flex items-center justify-between gap-3 px-3 pt-2 pb-3">
        <span className="text-muted-foreground flex min-w-0 items-center gap-1.5 px-1 text-xs">
          {editing ? (
            <span className="hidden sm:inline">Esc cancela</span>
          ) : (
            <>
              <Users aria-hidden="true" className="size-3.5 shrink-0" />
              <span className="truncate">Visível para todos</span>
            </>
          )}
        </span>
        <div className="flex shrink-0 items-center gap-2">
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
          {editing ? (
            <>
              <Button
                disabled={isSubmitting}
                onClick={props.onCancel}
                size="sm"
                type="button"
                variant="ghost"
              >
                Cancelar
              </Button>
              <Button disabled={isSubmitting} size="sm" type="submit">
                {isSubmitting ? 'Salvando…' : 'Salvar'}
              </Button>
            </>
          ) : (
            <Button
              className={cn(
                'rounded-full',
                !ready && 'bg-muted text-muted-foreground hover:bg-muted'
              )}
              aria-label={isSubmitting ? 'Publicando…' : 'Publicar'}
              disabled={isSubmitting}
              size="icon"
              title="Publicar (Ctrl + Enter)"
              type="submit"
            >
              <ArrowUp aria-hidden="true" />
            </Button>
          )}
        </div>
      </div>
    </form>
  )
}
