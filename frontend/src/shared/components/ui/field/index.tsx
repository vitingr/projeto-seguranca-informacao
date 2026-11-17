import { cn } from 'cn'
import type { ComponentProps } from 'react'

/** A labelled input with its error message, wired for screen readers. */
export const Field = ({
  className,
  error,
  id,
  label,
  ...props
}: ComponentProps<'input'> & { error?: string; id: string; label: string }) => {
  const errorId = `${id}-error`
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-sm font-medium" htmlFor={id}>
        {label}
      </label>
      <input
        className={cn(
          'border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive h-9 rounded-lg border px-3 text-sm outline-none focus-visible:ring-3',
          className
        )}
        aria-describedby={error ? errorId : undefined}
        aria-invalid={error ? true : undefined}
        id={id}
        {...props}
      />
      {error ? (
        <p className="text-destructive text-sm" id={errorId}>
          {error}
        </p>
      ) : null}
    </div>
  )
}

/** A form-level message (error or success) announced to screen readers. */
export const FormMessage = ({
  children,
  tone
}: {
  children: string
  tone: 'error' | 'success'
}) => (
  <p
    className={cn(
      'rounded-lg border px-3 py-2 text-sm',
      tone === 'error'
        ? 'border-destructive/40 text-destructive'
        : 'border-border text-foreground'
    )}
    data-testid={`form-${tone}`}
    role={tone === 'error' ? 'alert' : 'status'}
  >
    {children}
  </p>
)
