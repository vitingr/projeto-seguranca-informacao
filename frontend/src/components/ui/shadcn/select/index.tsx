'use client'

import { cn } from 'cn'
import { CheckIcon, ChevronDownIcon } from 'lucide-react'

import { Select as SelectPrimitive } from '@base-ui/react/select'

import type {
  SelectContentProps,
  SelectItemProps,
  SelectTriggerProps
} from './types'

const Select = SelectPrimitive.Root

function SelectValue({ className, ...props }: SelectPrimitive.Value.Props) {
  return (
    <SelectPrimitive.Value
      className={cn('flex flex-1 text-left', className)}
      data-slot="select-value"
      {...props}
    />
  )
}

function SelectTrigger({
  children,
  className,
  size = 'default',
  ...props
}: SelectTriggerProps) {
  return (
    <SelectPrimitive.Trigger
      className={cn(
        "border-input focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 data-placeholder:text-muted-foreground flex w-full items-center justify-between gap-1.5 rounded-sm border bg-transparent py-2 pr-2 pl-3 text-sm whitespace-nowrap transition-colors outline-none select-none focus-visible:ring-3 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:ring-3 data-[size=default]:h-9 data-[size=sm]:h-8 *:data-[slot=select-value]:line-clamp-1 *:data-[slot=select-value]:flex *:data-[slot=select-value]:items-center *:data-[slot=select-value]:gap-1.5 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        className
      )}
      data-size={size}
      data-slot="select-trigger"
      {...props}
    >
      {children}
      {/* A child, not `render`: Base UI's default "▼" text would reach the name */}
      <SelectPrimitive.Icon className="flex">
        <ChevronDownIcon className="text-muted-foreground pointer-events-none size-4" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  )
}

function SelectContent({
  align = 'start',
  alignItemWithTrigger = false,
  alignOffset = 0,
  children,
  className,
  side = 'bottom',
  sideOffset = 4,
  ...props
}: SelectContentProps) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Positioner
        align={align}
        alignItemWithTrigger={alignItemWithTrigger}
        alignOffset={alignOffset}
        className="isolate z-50"
        side={side}
        sideOffset={sideOffset}
      >
        <SelectPrimitive.Popup
          className={cn(
            'bg-popover text-popover-foreground border-border motion-safe:data-open:animate-in motion-safe:data-open:fade-in-0 motion-safe:data-closed:animate-out motion-safe:data-closed:fade-out-0 relative isolate z-50 max-h-(--available-height) w-(--anchor-width) min-w-36 origin-(--transform-origin) overflow-x-hidden overflow-y-auto rounded-sm border p-1 duration-100',
            className
          )}
          data-slot="select-content"
          {...props}
        >
          <SelectPrimitive.List>{children}</SelectPrimitive.List>
        </SelectPrimitive.Popup>
      </SelectPrimitive.Positioner>
    </SelectPrimitive.Portal>
  )
}

function SelectItem({ children, className, ...props }: SelectItemProps) {
  return (
    <SelectPrimitive.Item
      className={cn(
        'focus:bg-muted data-highlighted:bg-muted relative flex w-full cursor-default items-start gap-1.5 rounded-sm py-2 pr-8 pl-2 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50',
        className
      )}
      data-slot="select-item"
      {...props}
    >
      {children}
      <SelectPrimitive.ItemIndicator
        render={
          <span className="pointer-events-none absolute top-2.5 right-2 flex size-4 items-center justify-center" />
        }
      >
        <CheckIcon className="pointer-events-none size-4" />
      </SelectPrimitive.ItemIndicator>
    </SelectPrimitive.Item>
  )
}

const SelectItemText = SelectPrimitive.ItemText

export {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue
}
