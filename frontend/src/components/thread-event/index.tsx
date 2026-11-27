import { StatusBadge } from '@/components/status-badge'
import { ThreadTime } from '@/components/thread-message'

import type { ThreadEventProps } from './types'

export const EVENT_TEXTS = {
  changed: 'alterou o status para',
  opened: 'abriu o ticket'
} as const

export const ThreadEvent = ({ actor, now, status, time }: ThreadEventProps) => (
  <li
    className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 gap-y-1 pl-8 text-sm"
    data-event={status ? 'status_change' : 'opened'}
  >
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
