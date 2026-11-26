import { cn } from 'cn'

import type { AlertPartProps, AlertProps } from './types'
import { alertVariants } from './variance'

// shadcn alert (base-nova), adapted to the design rules: radius `sm`. The
// role is a prop, so a polite status can use it too.

function Alert({ className, role = 'alert', variant, ...props }: AlertProps) {
  return (
    <div
      className={cn(alertVariants({ variant }), className)}
      data-slot="alert"
      role={role}
      {...props}
    />
  )
}

function AlertTitle({ className, ...props }: AlertPartProps) {
  return (
    <div
      className={cn(
        '[&_a]:hover:text-foreground font-medium group-has-[>svg]/alert:col-start-2 [&_a]:underline [&_a]:underline-offset-3',
        className
      )}
      data-slot="alert-title"
      {...props}
    />
  )
}

function AlertDescription({ className, ...props }: AlertPartProps) {
  return (
    <div
      className={cn(
        'text-muted-foreground [&_a]:hover:text-foreground text-sm text-balance md:text-pretty [&_a]:underline [&_a]:underline-offset-3',
        className
      )}
      data-slot="alert-description"
      {...props}
    />
  )
}

function AlertAction({ className, ...props }: AlertPartProps) {
  return (
    <div
      className={cn('absolute top-2 right-2', className)}
      data-slot="alert-action"
      {...props}
    />
  )
}

export { Alert, AlertAction, AlertDescription, AlertTitle }
