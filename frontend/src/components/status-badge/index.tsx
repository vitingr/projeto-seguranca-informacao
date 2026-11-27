import { cn } from 'cn'

import { Badge } from '@/components/ui/shadcn/badge'
import { TICKET_STATUS_LABELS } from '@/domain/tickets'
import type { TicketStatus } from '@/domain/tickets/types'

import type { StatusBadgeProps } from './types'

export const STATUS_BADGE_CLASSES: Record<TicketStatus, string> = {
  aberto: 'bg-status-aberto text-status-aberto-foreground',
  em_andamento: 'bg-status-em-andamento text-status-em-andamento-foreground',
  fechado: 'bg-status-fechado text-status-fechado-foreground',
  resolvido: 'bg-status-resolvido text-status-resolvido-foreground'
}

export function StatusBadge({ className, status }: StatusBadgeProps) {
  return (
    <Badge
      className={cn(STATUS_BADGE_CLASSES[status], className)}
      data-status={status}
    >
      {TICKET_STATUS_LABELS[status]}
    </Badge>
  )
}
