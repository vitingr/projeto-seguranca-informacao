import { cn } from 'cn'
import Link from 'next/link'

import { buttonVariants } from '@/components/ui/shadcn/button'

import type { EmptyStateProps } from './types'

/**
 * An empty list (spec 003 AC-1.8, AC-4.6): a quiet bordered box with its
 * text and the one useful next action.
 */
export const EmptyState = ({ action, text }: EmptyStateProps) => (
  <div
    className="border-border flex flex-col items-start gap-3 rounded-sm border px-4 py-8 sm:items-center sm:text-center"
    data-testid="empty-state"
  >
    <p className="text-muted-foreground text-sm">{text}</p>
    {action ? (
      <Link
        className={cn(buttonVariants({ variant: 'outline' }))}
        href={action.href}
      >
        {action.label}
      </Link>
    ) : null}
  </div>
)
