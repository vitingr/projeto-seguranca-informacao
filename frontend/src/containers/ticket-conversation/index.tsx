'use client'

import { cn } from 'cn'
import { RotateCwIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useRef, useTransition } from 'react'

import { Separator } from '@/components/ui/shadcn/separator'
import { formatClock } from '@/domain/date-time'

import type { TicketConversationProps } from './types'

export const CONVERSATION_TEXTS = {
  refresh: 'Atualizar',
  refreshing: 'Atualizando…',
  region: 'Mensagens do ticket'
} as const

/** "1 mensagem", "N mensagens" (status lines are not messages). */
export const messageCountText = (count: number) =>
  count === 1 ? '1 mensagem' : `${count} mensagens`

/** "Atualizado às 14:12", from the server's render time (D-4). */
export const refreshedAtText = (iso: string) => {
  const clock = formatClock(iso)
  return clock ? `Atualizado às ${clock}` : 'Atualizado agora'
}

/** How close to the end still counts as "at the latest entry", in px. */
const NEAR_END = 48

/**
 * The "Conversa" panel (spec 003 AC-3.1 to AC-3.4a, AC-3.10, D-4, plan 6.5):
 * a header row with the message count and "Atualizado às HH:MM ·
 * Atualizar", a hairline under it, then the thread in a bounded scroll
 * region (thin scrollbar) that opens on its latest entry. Nothing refreshes
 * by itself (BR-9): "Atualizar" renders the page again on the server; the
 * view stays at the end if the user was there, otherwise where it was.
 */
export const TicketConversation = ({
  children,
  footer,
  messageCount,
  renderedAt
}: TicketConversationProps) => {
  const router = useRouter()
  const [pending, startTransition] = useTransition()
  const scroller = useRef<HTMLDivElement>(null)
  const atEnd = useRef(true)

  // The panel may start hidden (the "Detalhes" tab keeps it mounted), so the
  // jump to the end runs whenever the region or its content changes size:
  // when the tab opens, and when a refresh brings new entries.
  useEffect(() => {
    const element = scroller.current
    if (!element) return
    const toEnd = () => {
      if (atEnd.current) {
        element.scrollTo({ behavior: 'instant', top: element.scrollHeight })
      }
    }
    toEnd()
    const observer = new ResizeObserver(toEnd)
    observer.observe(element)
    for (const child of element.children) observer.observe(child)
    return () => observer.disconnect()
  }, [renderedAt])

  const refresh = () => {
    if (pending) return
    startTransition(() => {
      router.refresh()
    })
  }

  return (
    <div
      className="border-border flex flex-col rounded-sm border"
      data-testid="ticket-conversation"
    >
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-2.5 text-sm">
        <p className="font-medium" data-testid="message-count">
          {messageCountText(messageCount)}
        </p>
        <p className="text-muted-foreground flex items-center gap-1.5">
          <span data-testid="refreshed-at">{refreshedAtText(renderedAt)}</span>
          <span aria-hidden>·</span>
          <button
            aria-disabled={pending || undefined}
            className="text-foreground focus-visible:ring-ring/50 inline-flex items-center gap-1 rounded-sm font-medium underline-offset-4 outline-none hover:underline focus-visible:ring-3"
            onClick={refresh}
            type="button"
          >
            <RotateCwIcon
              className={cn('size-3.5', pending && 'motion-safe:animate-spin')}
              aria-hidden
            />
            {pending
              ? CONVERSATION_TEXTS.refreshing
              : CONVERSATION_TEXTS.refresh}
          </button>
        </p>
      </div>
      <Separator />
      <div
        onScroll={event => {
          const element = event.currentTarget
          atEnd.current =
            element.scrollHeight - element.scrollTop - element.clientHeight <
            NEAR_END
        }}
        aria-label={CONVERSATION_TEXTS.region}
        className="thread-scroll overflow-y-auto px-4 py-4"
        data-testid="thread-scroll"
        ref={scroller}
        role="region"
        tabIndex={0}
      >
        {children}
      </div>
      {footer}
    </div>
  )
}
