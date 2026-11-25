import { cn } from 'cn'
import {
  BugIcon,
  FileIcon,
  KeyRoundIcon,
  MessageSquareIcon,
  RocketIcon
} from 'lucide-react'

import type { TicketTypeIconProps } from './types'

const ICONS = {
  acesso: KeyRoundIcon,
  bug: BugIcon,
  deploy: RocketIcon,
  outro: MessageSquareIcon
} as const

/**
 * The icon of a ticket type in a small muted tile (type cards and the
 * preview). Decorative: the type label is always written next to it.
 */
export const TicketTypeIcon = ({
  className,
  selected = false,
  type
}: TicketTypeIconProps) => {
  const Icon = type ? ICONS[type] : FileIcon
  return (
    <span
      className={cn(
        'flex size-9 shrink-0 items-center justify-center rounded-sm transition-colors motion-reduce:transition-none',
        selected
          ? 'bg-primary text-primary-foreground'
          : 'bg-muted text-foreground',
        className
      )}
      aria-hidden
    >
      <Icon className="size-4" />
    </span>
  )
}
