import { cn } from 'cn'

import type { EyebrowProps } from './types'

// Small label above titles, marked with the accent.
export const Eyebrow: React.FC<EyebrowProps> = ({ children, className }) => (
  <span
    className={cn(
      'text-brand-ink flex w-fit items-center gap-2 text-xs font-semibold tracking-widest uppercase',
      className
    )}
  >
    <span aria-hidden="true" className="bg-brand h-0.5 w-4 rounded-full" />
    {children}
  </span>
)
