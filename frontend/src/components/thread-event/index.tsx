import { StatusBadge } from '@/components/status-badge'
import { ThreadTime } from '@/components/thread-message'

import type { ThreadEventProps } from './types'

export const EVENT_TEXTS = {
  changed: 'alterou o status para',
  opened: 'abriu o ticket'
} as const

/**
 * An event line of a thread, in the style of "Orchestrator started Coder"
 * of `referencia-conversa.mp4` (spec 003 AC-3.2, AC-5.3): no card, small and
 * muted, the actor in dark text, aligned with the cards, then the new
 * status badge and the time: "Suporte alterou o status para [Em andamento]
 * · 14:06", "Você abriu o ticket · 14:02".
 */
export const ThreadEvent = ({ actor, now, status, time }: ThreadEventProps) => (
  <li
    className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 gap-y-1 pl-8 text-sm"
    data-event={status ? 'status_change' : 'opened'}
  >
    {/* The spaces are real text, so the line reads as a sentence. */}
    <span className="text-foreground font-medium break-all">{actor}</span>{' '}
    <span>{status ? EVENT_TEXTS.changed : EVENT_TEXTS.opened}</span>{' '}
    {status ? (
      <>
        <StatusBadge status={status} />{' '}
      </>
    ) : null}
    <span aria-hidden>·</span>{' '}
    <span className="text-xs">
      <ThreadTime iso={time} now={now} />
    </span>
  </li>
)
