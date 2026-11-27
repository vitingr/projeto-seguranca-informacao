import type { ReactNode } from 'react'

import type { ApiClient, ThreadScope } from '@/repositories/api/types'

/** The reply box of the page; absent where nobody may reply. */
export interface ReplySettings {
  /** The ticket is "Fechado": the closed line replaces the box (AC-3.8). */
  closed: boolean
  /** "Novo ticket" under the closed line (the author's page only). */
  closedAction?: { href: string; label: string }
  placeholder: string
  scope: ThreadScope
  ticketId: number
  /** "Visível para a equipe de suporte" or "Visível para <login>". */
  trayText: string
}

export interface TicketConversationProps {
  /** The thread entries, rendered by the server. */
  children: ReactNode
  /** The messages of the thread (status lines are not counted). */
  messageCount: number
  /** When the server rendered the thread (ISO 8601): "Atualizado às". */
  renderedAt: string
  reply?: ReplySettings
}

export interface ReplyDeps {
  send: ApiClient['sendMessage']
}

export interface ReplyInput {
  /** The text as typed; it is trimmed before sending. */
  raw: string
  scope: ThreadScope
  ticketId: number
}

export type ReplyOutcome =
  | { entryId: null | number; kind: 'sent' }
  | { kind: 'failed'; message: string; refresh: boolean }
  | { kind: 'invalid'; message: string }
