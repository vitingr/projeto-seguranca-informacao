import type { ThreadEntry } from '@/domain/tickets/types'

export type ThreadView = 'admin' | 'own'

export interface ThreadProps {
  authorLogin?: string
  entries: readonly ThreadEntry[]
  now: string
  openedAt: string
  view: ThreadView
}
