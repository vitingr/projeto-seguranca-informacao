import { cn } from 'cn'

function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      className={cn(
        'border-input bg-card placeholder:text-muted-foreground hover:border-foreground/30 focus-visible:border-ring focus-visible:ring-ring/25 disabled:bg-input/50 aria-invalid:border-destructive aria-invalid:ring-destructive/15 flex field-sizing-content min-h-24 w-full rounded-xl border px-3.5 py-3 text-base leading-relaxed outline-none focus-visible:ring-4 disabled:cursor-not-allowed disabled:opacity-50 motion-safe:transition-[border-color,box-shadow] md:text-sm',
        className
      )}
      data-slot="textarea"
      {...props}
    />
  )
}

export { Textarea }
