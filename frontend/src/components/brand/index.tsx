import { cn } from 'cn'

import type { BrandProps } from './types'

// Logo mark: the initials on ink, with the accent as a corner tab.
export const BrandMark: React.FC<BrandProps> = ({ className }) => (
  <span
    className={cn(
      'bg-primary text-primary-foreground relative flex size-8 shrink-0 items-center justify-center rounded-lg text-xs font-semibold tracking-tight',
      className
    )}
    aria-hidden="true"
  >
    SI
    <span className="bg-brand ring-background absolute -top-0.5 -right-0.5 size-2.5 rounded-xs ring-2" />
  </span>
)

export const Brand: React.FC<BrandProps> = ({ className }) => (
  <span className={cn('flex items-center gap-3', className)}>
    <BrandMark />
    <span className="text-sm font-semibold tracking-tight">
      Sistemas da Informação
    </span>
  </span>
)
