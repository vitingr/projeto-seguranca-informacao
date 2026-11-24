import { cn } from 'cn'
import { ChevronDownIcon } from 'lucide-react'

import type { NativeSelectProps } from './types'

function NativeSelect({
  className,
  size = 'default',
  ...props
}: NativeSelectProps) {
  return (
    <div
      className={cn(
        'group/native-select relative w-fit has-[select:disabled]:opacity-50',
        className
      )}
      data-size={size}
      data-slot="native-select-wrapper"
    >
      <select
        className="border-input bg-card placeholder:text-muted-foreground hover:border-foreground/30 focus-visible:border-ring focus-visible:ring-ring/25 aria-invalid:border-destructive aria-invalid:ring-destructive/15 h-8 w-full min-w-0 appearance-none rounded-lg border py-1 pr-8 pl-2.5 text-sm outline-none select-none focus-visible:ring-3 disabled:pointer-events-none disabled:cursor-not-allowed data-[size=lg]:h-11 data-[size=lg]:rounded-xl data-[size=lg]:pr-10 data-[size=lg]:pl-3.5 data-[size=lg]:focus-visible:ring-4 data-[size=sm]:h-7 data-[size=sm]:py-0.5 motion-safe:transition-[border-color,box-shadow]"
        data-size={size}
        data-slot="native-select"
        {...props}
      />
      <ChevronDownIcon
        aria-hidden="true"
        className="text-muted-foreground pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 select-none"
        data-slot="native-select-icon"
      />
    </div>
  )
}

function NativeSelectOption(props: React.ComponentProps<'option'>) {
  return <option data-slot="native-select-option" {...props} />
}

function NativeSelectOptGroup(props: React.ComponentProps<'optgroup'>) {
  return <optgroup data-slot="native-select-optgroup" {...props} />
}

export { NativeSelect, NativeSelectOptGroup, NativeSelectOption }
