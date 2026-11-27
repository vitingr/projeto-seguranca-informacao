import { cn } from 'cn'
import { InfoIcon } from 'lucide-react'

import { Textarea } from '@/components/ui/shadcn/textarea'

import type { TextAreaFieldProps } from './types'

export const limitText = (max: number) => `Até ${max} caracteres.`

export const TextAreaField = ({
  'aria-describedby': describedBy,
  className,
  count,
  hint,
  id,
  max,
  ...props
}: TextAreaFieldProps) => {
  const hintId = `${id}-strip`
  const limitId = `${id}-limit`
  const over = count > max

  return (
    <div
      className={cn(
        'border-input focus-within:border-ring focus-within:ring-ring/50 has-aria-invalid:border-destructive has-aria-invalid:ring-destructive/20 flex flex-col overflow-hidden rounded-sm border transition-colors focus-within:ring-3 has-aria-invalid:ring-3 motion-reduce:transition-none',
        className
      )}
      data-slot="text-area-field"
    >
      <Textarea
        aria-describedby={
          [describedBy, hint ? hintId : null, limitId]
            .filter(Boolean)
            .join(' ') || undefined
        }
        className="max-h-72 min-h-24 resize-none rounded-none border-0 px-3 pt-3 pb-1 focus-visible:ring-0 aria-invalid:ring-0"
        id={id}
        {...props}
      />
      <div className="flex items-center justify-end px-3 pb-2">
        <span
          className={cn(
            'text-xs tabular-nums',
            over ? 'text-destructive' : 'text-muted-foreground'
          )}
          data-testid={`${id}-counter`}
          aria-hidden
        >
          {count} / {max}
        </span>
        <span className="sr-only" id={limitId}>
          {limitText(max)}
        </span>
      </div>
      {hint ? (
        <p
          className="border-border bg-muted text-muted-foreground flex items-start gap-2 border-t px-3 py-2 text-xs"
          id={hintId}
        >
          <InfoIcon className="size-3.5 shrink-0 translate-y-px" aria-hidden />
          {hint}
        </p>
      ) : null}
    </div>
  )
}
