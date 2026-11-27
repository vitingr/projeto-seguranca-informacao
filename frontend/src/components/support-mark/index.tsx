import { cn } from 'cn'
import { HeadsetIcon } from 'lucide-react'

import type { SupportMarkProps } from './types'

/**
 * The one mark of the support team in a thread (spec 003 section 7): a
 * small neutral square tile with a headset, in front of every admin message
 * the comum user sees. Decorative: the name next to it says "Suporte".
 */
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
