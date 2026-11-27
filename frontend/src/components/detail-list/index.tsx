import { cn } from 'cn'

import type { DetailListProps, DetailRowProps } from './types'

export const DetailList = ({ children }: DetailListProps) => (
  <dl className="border-border divide-border flex flex-col divide-y rounded-sm border">
    {children}
  </dl>
)

export const DetailRow = ({
  children,
  label,
  preserveLines = false
}: DetailRowProps) => (
  <div className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:gap-6">
    <dt className="text-muted-foreground text-sm sm:w-44 sm:shrink-0">
      {label}
    </dt>
    <dd
      className={cn(
        'min-w-0 flex-1 text-sm break-words',
        preserveLines && 'whitespace-pre-wrap'
      )}
    >
      {children}
    </dd>
  </div>
)
