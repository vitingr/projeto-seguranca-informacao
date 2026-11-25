import { cn } from 'cn'

import type { TextareaProps } from './types'

// shadcn textarea (base-nova), adapted to the design rules: radius `sm`, no
// dark variants.

function Textarea({ className, ...props }: TextareaProps) {
  return (
    <textarea
      className={cn(
        'border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 disabled:bg-input/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 flex field-sizing-content min-h-16 w-full rounded-sm border bg-transparent px-2.5 py-2 text-base transition-colors outline-none focus-visible:ring-3 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:ring-3 md:text-sm',
        className
      )}
      data-slot="textarea"
      {...props}
    />
  )
}

export { Textarea }
