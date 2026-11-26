'use client'

import { cn } from 'cn'

import { Separator as SeparatorPrimitive } from '@base-ui/react/separator'

import type { SeparatorProps } from './types'

// shadcn separator (base-nova), unchanged apart from the module layout.

function Separator({
  className,
  orientation = 'horizontal',
  ...props
}: SeparatorProps) {
  return (
    <SeparatorPrimitive
      className={cn(
        'bg-border shrink-0 data-horizontal:h-px data-horizontal:w-full data-vertical:w-px data-vertical:self-stretch',
        className
      )}
      data-slot="separator"
      orientation={orientation}
      {...props}
    />
  )
}

export { Separator }
