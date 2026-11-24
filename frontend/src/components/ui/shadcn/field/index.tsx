import { cn } from 'cn'
import { CircleAlert, CircleCheck } from 'lucide-react'

import type { FieldProps, FormMessageProps } from './types'

export const Field = ({
  className,
  error,
  hint,
  id,
  label,
  ...props
}: FieldProps) => {
  const errorId = `${id}-error`
  const hintId = `${id}-hint`
  const describedBy = error ? errorId : hint ? hintId : undefined
  return (
    <div className="flex flex-col gap-2">
      <label className="text-sm font-medium" htmlFor={id}>
        {label}
      </label>
      <input
        className={cn(
          'border-input bg-card placeholder:text-muted-foreground hover:border-foreground/30 focus-visible:border-ring focus-visible:ring-ring/25 aria-invalid:border-destructive aria-invalid:ring-destructive/15 h-11 rounded-xl border px-3.5 text-sm outline-none focus-visible:ring-4 motion-safe:transition-[border-color,box-shadow]',
          className
        )}
        aria-describedby={describedBy}
        aria-invalid={error ? true : undefined}
        id={id}
        {...props}
      />
      {error ? (
        <p
          className="text-destructive motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-top-1 flex items-start gap-1.5 text-sm"
          id={errorId}
        >
          <CircleAlert
            aria-hidden="true"
            className="size-4 shrink-0 translate-y-0.5"
          />
          {error}
        </p>
      ) : hint ? (
        <p className="text-muted-foreground text-sm text-pretty" id={hintId}>
          {hint}
        </p>
      ) : null}
    </div>
  )
}

/** A form-level message (error or success) announced to screen readers. */
export const FormMessage = ({ children, tone }: FormMessageProps) => {
  const Icon = tone === 'error' ? CircleAlert : CircleCheck
  return (
    <p
      className={cn(
        'motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-top-1 flex items-start gap-2.5 rounded-xl border px-3.5 py-3 text-sm',
        tone === 'error'
          ? 'bg-destructive-soft border-destructive/20 text-destructive'
          : 'bg-success-soft border-success/20 text-success'
      )}
      data-testid={`form-${tone}`}
      role={tone === 'error' ? 'alert' : 'status'}
    >
      <Icon aria-hidden="true" className="size-4 shrink-0 translate-y-0.5" />
      {children}
    </p>
  )
}
