import { Avatar } from '@/components/avatar'
import { SupportMark } from '@/components/support-mark'
import { ThreadEvent } from '@/components/thread-event'
import { ThreadMessage } from '@/components/thread-message'
import type { ThreadEntry } from '@/domain/tickets/types'

import type { ThreadProps, ThreadView } from './types'

export const THREAD_TEXTS = {
  empty:
    'Nenhuma mensagem ainda. Escreva abaixo se quiser acrescentar algo ao ticket.',
  label: 'Conversa do ticket',
  support: 'Suporte',
  you: 'Você'
} as const

/**
 * Who wrote an entry, as the viewer sees it (plan P-11, spec Q-6): "Você"
 * for the viewer; for the comum author every admin is "Suporte"; for an
 * admin, the login of whoever acted.
 */
export const actorName = (entry: ThreadEntry, view: ThreadView): string => {
  if (entry.mine) return THREAD_TEXTS.you
  if (view === 'own') return THREAD_TEXTS.support
  return entry.actorLogin ?? THREAD_TEXTS.support
}

/**
 * The entries of a ticket thread (spec 003 AC-3.1 to AC-3.3, AC-5.3), oldest
 * first: the opening line, then the messages and the status lines in date
 * order. The other side is on the left with its mark (the support mark for
 * the comum user; for an admin, the author's generated avatar, and the
 * support mark for other admins); own messages are on the right. Everything
 * is plain text (S-10). With no entry yet, a muted line invites the first
 * message.
 */
export const Thread = ({
  authorLogin,
  entries,
  now,
  openedAt,
  view
}: ThreadProps) => {
  const opener =
    view === 'own' ? THREAD_TEXTS.you : (authorLogin ?? THREAD_TEXTS.support)
  return (
    <ol aria-label={THREAD_TEXTS.label} className="flex flex-col gap-4">
      <ThreadEvent actor={opener} now={now} time={openedAt} />
      {entries.length === 0 ? (
        <li className="text-muted-foreground pl-8 text-sm" data-empty>
          {THREAD_TEXTS.empty}
        </li>
      ) : null}
      {entries.map(entry => {
        const name = actorName(entry, view)
        if (entry.kind === 'status_change') {
          return (
            <ThreadEvent
              actor={name}
              key={entry.id}
              now={now}
              status={entry.to}
              time={entry.createdAt}
            />
          )
        }
        const mark =
          view === 'admin' && entry.side === 'author' ? (
            <Avatar
              className="size-6 [&>span]:text-xs"
              login={entry.actorLogin ?? ''}
            />
          ) : (
            <SupportMark />
          )
        return (
          <ThreadMessage
            key={entry.id}
            mark={mark}
            name={name}
            now={now}
            own={entry.mine}
            text={entry.text}
            time={entry.createdAt}
          />
        )
      })}
    </ol>
  )
}
