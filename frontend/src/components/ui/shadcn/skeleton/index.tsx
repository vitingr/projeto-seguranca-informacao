import { cn } from 'cn'

import type { SkeletonProps } from './types'

function Skeleton({ className, ...props }: SkeletonProps) {
  return (
    <div
      className={cn('bg-muted rounded-sm motion-safe:animate-pulse', className)}
      data-slot="skeleton"
      {...props}
    />
  )
}

export { Skeleton }
