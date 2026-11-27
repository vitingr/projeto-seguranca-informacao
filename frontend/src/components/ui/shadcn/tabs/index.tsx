'use client'

import { cn } from 'cn'

import { Tabs as TabsPrimitive } from '@base-ui/react/tabs'

import type {
  TabsContentProps,
  TabsListProps,
  TabsProps,
  TabsTriggerProps
} from './types'

function Tabs({ className, orientation = 'horizontal', ...props }: TabsProps) {
  return (
    <TabsPrimitive.Root
      className={cn(
        'group/tabs flex gap-2 data-[orientation=horizontal]:flex-col',
        className
      )}
      data-orientation={orientation}
      data-slot="tabs"
      orientation={orientation}
      {...props}
    />
  )
}

function TabsList({ className, ...props }: TabsListProps) {
  return (
    <TabsPrimitive.List
      className={cn(
        'bg-muted text-muted-foreground inline-flex w-fit items-center justify-center gap-0.5 rounded-sm p-0.5 group-data-[orientation=vertical]/tabs:h-fit group-data-[orientation=vertical]/tabs:flex-col',
        className
      )}
      data-slot="tabs-list"
      {...props}
    />
  )
}

function TabsTrigger({ className, ...props }: TabsTriggerProps) {
  return (
    <TabsPrimitive.Tab
      className={cn(
        'text-muted-foreground hover:text-foreground focus-visible:border-ring focus-visible:ring-ring/50 data-active:bg-background data-active:text-foreground data-active:border-border inline-flex h-8 flex-1 items-center justify-center gap-1.5 rounded-sm border border-transparent px-3 text-sm font-medium whitespace-nowrap transition-colors focus-visible:ring-3 focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4',
        className
      )}
      data-slot="tabs-trigger"
      {...props}
    />
  )
}

function TabsContent({ className, ...props }: TabsContentProps) {
  return (
    <TabsPrimitive.Panel
      className={cn(
        'focus-visible:ring-ring/50 flex-1 rounded-sm text-sm outline-none focus-visible:ring-3',
        className
      )}
      data-slot="tabs-content"
      {...props}
    />
  )
}

export { Tabs, TabsContent, TabsList, TabsTrigger }
