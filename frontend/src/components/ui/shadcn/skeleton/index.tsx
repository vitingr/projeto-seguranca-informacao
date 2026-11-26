import { cn } from 'cn'

import type { SkeletonProps } from './types'

// shadcn skeleton (base-nova), adapted to the design rules: radius `sm`, and
// the pulse only when the user allows motion.

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
