import type { ReactNode } from 'react'

import type { ApiClient, ThreadScope } from '@/repositories/api/types'

export interface ReplySettings {
  closed: boolean
  closedAction?: { href: string; label: string }
  placeholder: string
  scope: ThreadScope
  ticketId: number
  trayText: string
}

export interface TicketConversationProps {
  children: ReactNode
  messageCount: number
  renderedAt: string
  reply?: ReplySettings
}

export interface ReplyDeps {
  send: ApiClient['sendMessage']
}

export interface ReplyInput {
  raw: string
  scope: ThreadScope
  ticketId: number
}

export type ReplyOutcome =
  | { entryId: null | number; kind: 'sent' }
  | { kind: 'failed'; message: string; refresh: boolean }
  | { kind: 'invalid'; message: string }
