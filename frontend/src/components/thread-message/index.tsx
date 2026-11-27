import { cn } from 'cn'

import { formatDateTime, formatThreadTime } from '@/domain/date-time'

import type { ThreadMessageProps, ThreadTimeProps } from './types'

export const ThreadTime = ({ iso, now }: ThreadTimeProps) => {
  const full = formatDateTime(iso) ?? '—'
  return (
    <time dateTime={iso} title={full}>
      <span aria-hidden>{formatThreadTime(iso, now) ?? '—'}</span>
      <span className="sr-only">{full}</span>
    </time>
  )
}

export const ThreadMessage = ({
  entryId,
  mark,
  name,
  now,
  own,
  text,
  time
}: ThreadMessageProps) => (
  <li
    className={cn('flex items-start gap-2', own && 'justify-end')}
    data-entry-id={entryId}
    data-side={own ? 'own' : 'other'}
  >
    {own ? null : <span className="flex pt-2.5">{mark}</span>}
    <article
      className={cn(
        'border-border hover:border-foreground/30 flex max-w-4/5 min-w-0 flex-col gap-1 rounded-sm border px-4 py-3 transition-colors',
        own ? 'bg-muted' : 'bg-background'
      )}
    >
      <header className="flex items-baseline justify-between gap-4">
        <p className="text-sm font-medium break-all">{name}</p>
        <p className="text-muted-foreground shrink-0 text-xs">
          <ThreadTime iso={time} now={now} />
        </p>
      </header>
      <p className="text-sm break-words whitespace-pre-wrap">{text}</p>
    </article>
  </li>
)
