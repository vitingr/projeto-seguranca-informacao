import type { AuditEventType } from '@/domain/audit/types'

export interface AuditLogProps {
  /** The page of the log, already parsed from `?pagina=`. */
  page: number
  /** The event filter, already parsed from `?tipo=` (null: every event). */
  type: AuditEventType | null
}
