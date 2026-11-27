import { cn } from 'cn'

import type { FormFieldProps, FormInputProps } from './types'

export const FIELD_MARKERS = {
  optional: '(opcional)',
  required: '(obrigatório)'
} as const

export const fieldIds = (id: string) => ({
  error: `${id}-error`,
  hint: `${id}-hint`,
  label: `${id}-label`
})

export const FormField = ({
  children,
  className,
  error,
  hint,
  id,
  label,
  labelId,
  required
}: FormFieldProps) => {
  const ids = fieldIds(id)
  const describedBy =
    [hint ? ids.hint : null, error ? ids.error : null]
      .filter(Boolean)
      .join(' ') || undefined

  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <label
        className="flex items-baseline gap-1 text-sm font-medium"
        htmlFor={id}
        id={labelId ?? ids.label}
      >
        {label}
        {required ? (
          <>
            <span className="text-destructive" aria-hidden>
              *
            </span>
            <span className="sr-only">{FIELD_MARKERS.required}</span>
          </>
        ) : (
          <span className="text-muted-foreground font-normal">
            {FIELD_MARKERS.optional}
          </span>
        )}
      </label>
      {hint ? (
        <p className="text-muted-foreground text-xs" id={ids.hint}>
          {hint}
        </p>
      ) : null}
      {children({
        'aria-describedby': describedBy,
        'aria-invalid': error ? true : undefined,
        'aria-required': required ? true : undefined,
        id
      })}
      {error ? (
        <p className="text-destructive text-sm" id={ids.error}>
          {error}
        </p>
      ) : null}
    </div>
  )
}

export const FormInput = ({ className, ...props }: FormInputProps) => (
  <input
    className={cn(
      'border-input bg-background placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 h-9 w-full rounded-sm border px-3 text-sm outline-none focus-visible:ring-3 aria-invalid:ring-3',
      className
    )}
    {...props}
  />
)
