import { cn } from 'cn'
import { HeadsetIcon } from 'lucide-react'

import type { SupportMarkProps } from './types'

export const SupportMark = ({ className }: SupportMarkProps) => (
  <span
    className={cn(
      'bg-muted text-muted-foreground border-border inline-flex size-6 shrink-0 items-center justify-center rounded-sm border',
      className
    )}
    aria-hidden
    data-support-mark
  >
    <HeadsetIcon className="size-3.5" />
  </span>
)
