import type { ReactNode } from 'react'

import type { AuditEvent } from '@/domain/audit/types'

export interface AuditTableProps {
  events: AuditEvent[]
}

export interface AuditCellProps {
  children: ReactNode
  /** Shown before the value when the table is stacked (mobile). */
  label: string
  testId: string
}
