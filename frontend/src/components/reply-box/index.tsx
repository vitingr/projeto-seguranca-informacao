import { cn } from 'cn'
import { ArrowUpIcon, EyeIcon, LoaderCircleIcon, LockIcon } from 'lucide-react'
import Link from 'next/link'

import { buttonVariants } from '@/components/ui/shadcn/button'
import { Textarea } from '@/components/ui/shadcn/textarea'
import { MESSAGE_MAX, codePoints, trimField } from '@/domain/tickets'

import type { ReplyBoxProps } from './types'

export const REPLY_TEXTS = {
  closed:
    'Este ticket foi fechado. Abra um novo ticket se precisar de mais ajuda.',
  label: 'Mensagem',
  limit: `Até ${MESSAGE_MAX} caracteres.`,
  send: 'Enviar mensagem',
  shortcut: 'Ctrl+Enter para enviar',
  shortcutMac: '⌘+Enter para enviar'
} as const

export const hasText = (value: string) => trimField(value) !== ''

export const ReplyBox = ({
  closed = false,
  closedAction,
  error,
  id,
  mac = false,
  onChange,
  onSend,
  placeholder,
  sending = false,
  textareaRef,
  trayText,
  value
}: ReplyBoxProps) => {
  if (closed) {
    return (
      <div
        className="text-muted-foreground flex flex-col gap-3 px-4 py-4 text-sm sm:flex-row sm:items-center sm:justify-between"
        data-testid="reply-closed"
      >
        <p className="flex items-start gap-2">
          <LockIcon className="size-4 shrink-0 translate-y-0.5" aria-hidden />
          {REPLY_TEXTS.closed}
        </p>
        {closedAction ? (
          <Link
            className={cn(buttonVariants({ variant: 'outline' }), 'w-fit')}
            href={closedAction.href}
          >
            {closedAction.label}
          </Link>
        ) : null}
      </div>
    )
  }

  const count = codePoints(value)
  const over = count > MESSAGE_MAX
  const ready = hasText(value) && !sending
  const errorId = `${id}-error`
  const limitId = `${id}-limit`

  return (
    <div className="flex flex-col gap-1.5 p-3" data-testid="reply-box">
      <label className="sr-only" htmlFor={id}>
        {REPLY_TEXTS.label}
      </label>
      <div className="flex flex-col">
        <div
          className={cn(
            'border-input bg-background focus-within:border-ring focus-within:ring-ring/50 flex flex-col overflow-hidden rounded-sm border transition-colors focus-within:ring-3 motion-reduce:transition-none',
            error &&
              'border-destructive focus-within:border-destructive ring-destructive/20 ring-3'
          )}
        >
          <Textarea
            aria-describedby={[error ? errorId : null, limitId]
              .filter(Boolean)
              .join(' ')}
            onKeyDown={event => {
              if (event.key !== 'Enter') return
              if (!(event.ctrlKey || event.metaKey)) return
              event.preventDefault()
              if (ready) onSend()
            }}
            aria-invalid={error ? true : undefined}
            className="max-h-48 min-h-16 resize-none rounded-none border-0 px-3 pt-3 pb-1 focus-visible:ring-0 aria-invalid:ring-0"
            id={id}
            name="texto"
            onChange={event => onChange(event.target.value)}
            placeholder={placeholder}
            readOnly={sending}
            ref={textareaRef}
            value={value}
          />
          <div className="flex items-center justify-end gap-3 px-3 pb-2">
            <span
              className={cn(
                'text-xs tabular-nums',
                over ? 'text-destructive' : 'text-muted-foreground'
              )}
              data-testid="reply-counter"
              aria-hidden
            >
              {count} / {MESSAGE_MAX}
            </span>
            <span className="sr-only" id={limitId}>
              {REPLY_TEXTS.limit}
            </span>
            <button
              className={cn(
                'focus-visible:ring-ring/50 inline-flex size-8 shrink-0 items-center justify-center rounded-sm transition-colors outline-none focus-visible:ring-3',
                ready || sending
                  ? 'bg-primary text-primary-foreground hover:bg-primary/80'
                  : 'bg-muted text-muted-foreground'
              )}
              onClick={() => {
                if (ready) onSend()
              }}
              aria-busy={sending || undefined}
              aria-disabled={!ready || undefined}
              aria-label={REPLY_TEXTS.send}
              data-testid="reply-send"
              type="button"
            >
              {sending ? (
                <LoaderCircleIcon
                  className="size-4 motion-safe:animate-spin"
                  aria-hidden
                />
              ) : (
                <ArrowUpIcon className="size-4" aria-hidden />
              )}
            </button>
          </div>
        </div>
        <div className="border-border bg-muted text-muted-foreground flex flex-wrap items-center justify-between gap-x-4 gap-y-1 rounded-b-sm border border-t-0 px-3 py-2 text-xs">
          <span className="flex min-w-0 items-center gap-1.5">
            <EyeIcon className="size-3.5 shrink-0" aria-hidden />
            <span className="break-all">{trayText}</span>
          </span>
          <span>{mac ? REPLY_TEXTS.shortcutMac : REPLY_TEXTS.shortcut}</span>
        </div>
      </div>
      {error ? (
        <p className="text-destructive text-sm" id={errorId}>
          {error}
        </p>
      ) : null}
    </div>
  )
}
