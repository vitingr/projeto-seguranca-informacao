import type { ThreadEntry } from '@/domain/tickets/types'

/** `own`: the comum author's page; `admin`: the support side (AC-5.3). */
export type ThreadView = 'admin' | 'own'

export interface ThreadProps {
  /** Admin view: the author's login, for the opening line and the avatar. */
  authorLogin?: string
  /** Oldest first, as the backend answers them. */
  entries: readonly ThreadEntry[]
  /** The server's time of the page render. */
  now: string
  /** The ticket's `created_at`: the time of the opening line. */
  openedAt: string
  view: ThreadView
}
