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

export const NEW_ENTRY_CLASS = 'motion-safe:animate-enter-blur'

export const messageCountText = (count: number) =>
  count === 1 ? '1 mensagem' : `${count} mensagens`

export const refreshedAtText = (iso: string) => {
  const clock = formatClock(iso)
  return clock ? `Atualizado às ${clock}` : 'Atualizado agora'
}

const NEAR_END = 48 // px

const noSubscription = () => () => {}

const readMac = () => /Mac|iPhone|iPad/.test(navigator.platform)

const sendOnce = singleFlight(sendReply)

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
  const { leave, leaving, stay, stayRef } = useLeaveGuard(
    reply !== undefined && !reply.closed && hasText(text)
  )

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
