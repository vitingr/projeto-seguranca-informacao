'use client'

import { cn } from 'cn'
import { RotateCwIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import {
  useEffect,
  useId,
  useRef,
  useState,
  useSyncExternalStore,
  useTransition
} from 'react'

import { InlineAlert } from '@/components/inline-alert'
import { ReplyBox, hasText } from '@/components/reply-box'
import { Separator } from '@/components/ui/shadcn/separator'
import { LeaveGuardBar, useLeaveGuard } from '@/containers/leave-guard'
import { formatClock } from '@/domain/date-time'
import { api } from '@/repositories/api'
import { singleFlight } from '@/utils/single-flight'

import { sendReply } from './reply'
import type { TicketConversationProps } from './types'

export const CONVERSATION_TEXTS = {
  leave: 'Sair sem enviar',
  leaveBody: 'A mensagem que você escreveu será perdida.',
  leaveTitle: 'Sair sem enviar?',
  refresh: 'Atualizar',
  refreshing: 'Atualizando…',
  region: 'Mensagens do ticket',
  sent: 'Mensagem enviada.',
  stay: 'Continuar no ticket'
} as const

/** The fade-in of a new own message, only when motion is allowed. */
export const NEW_ENTRY_CLASS = 'motion-safe:animate-enter-blur'

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

const noSubscription = () => () => {}

/** macOS shows "⌘+Enter"; the server and other systems "Ctrl+Enter". */
const readMac = () => /Mac|iPhone|iPad/.test(navigator.platform)

/** One reply at a time: a second click while sending starts nothing. */
const sendOnce = singleFlight(sendReply)

/**
 * The "Conversa" panel (spec 003 AC-3.1 to AC-3.10, D-4, plan 6.5): a header
 * row with the message count and "Atualizado às HH:MM · Atualizar", a
 * hairline, the thread in a bounded scroll region that opens on its latest
 * entry, and the reply box pinned under it. Nothing refreshes by itself
 * (BR-9): "Atualizar" and a sent reply render the page again on the server.
 * The view stays at the end if the user was there, otherwise where it was;
 * after the user's own reply it goes to the end.
 */
export const TicketConversation = ({
  children,
  messageCount,
  renderedAt,
  reply
}: TicketConversationProps) => {
  const router = useRouter()
  const [pending, startTransition] = useTransition()
  const scroller = useRef<HTMLDivElement>(null)
  const atEnd = useRef(true)
  const newEntry = useRef<null | number>(null)
  const textareaId = useId()
  const [text, setText] = useState('')
  const [fieldError, setFieldError] = useState<string | undefined>()
  const [sendError, setSendError] = useState<null | string>(null)
  const [sending, setSending] = useState(false)
  const [announcement, setAnnouncement] = useState('')
  const mac = useSyncExternalStore(noSubscription, readMac, () => false)
  // AC-3.9: a typed reply not sent warns before the page is left (the text
  // lives only in this page, never in storage or the address, S-11).
  const { leave, leaving, stay, stayRef } = useLeaveGuard(
    reply !== undefined && !reply.closed && hasText(text)
  )

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
    if (newEntry.current !== null) {
      element
        .querySelector(`[data-entry-id="${newEntry.current}"]`)
        ?.classList.add(NEW_ENTRY_CLASS)
      newEntry.current = null
    }
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

  const send = async () => {
    if (!reply || sending) return
    setSendError(null)
    setSending(true)
    const outcome = await sendOnce(
      { send: api.sendMessage },
      { raw: text, scope: reply.scope, ticketId: reply.ticketId }
    )
    setSending(false)
    if (outcome === null) return
    if (outcome.kind === 'invalid') {
      setFieldError(outcome.message)
      return
    }
    if (outcome.kind === 'failed') {
      setSendError(outcome.message)
      if (outcome.refresh) refresh()
      return
    }
    setText('')
    setFieldError(undefined)
    setAnnouncement(CONVERSATION_TEXTS.sent)
    atEnd.current = true
    newEntry.current = outcome.entryId
    refresh()
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
      {reply ? (
        <>
          <Separator />
          <div className="px-3 pt-3 empty:hidden">
            <LeaveGuardBar
              body={CONVERSATION_TEXTS.leaveBody}
              leaveLabel={CONVERSATION_TEXTS.leave}
              onLeave={leave}
              onStay={stay}
              open={leaving !== null}
              stayLabel={CONVERSATION_TEXTS.stay}
              stayRef={stayRef}
              title={CONVERSATION_TEXTS.leaveTitle}
            />
          </div>
          {sendError ? (
            <div className="px-3 pt-3">
              <InlineAlert>{sendError}</InlineAlert>
            </div>
          ) : null}
          <ReplyBox
            onChange={value => {
              setText(value)
              if (fieldError) setFieldError(undefined)
            }}
            closed={reply.closed}
            closedAction={reply.closedAction}
            error={fieldError}
            id={textareaId}
            mac={mac}
            onSend={send}
            placeholder={reply.placeholder}
            sending={sending}
            trayText={reply.trayText}
            value={text}
          />
        </>
      ) : null}
      <p aria-live="polite" className="sr-only" data-testid="reply-status">
        {announcement}
      </p>
    </div>
  )
}
