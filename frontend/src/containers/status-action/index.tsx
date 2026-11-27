'use client'

import { cn } from 'cn'
import { CheckIcon, ChevronDownIcon, LoaderCircleIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useRef, useState, useTransition } from 'react'

import { InlineAlert } from '@/components/inline-alert'
import { Button, buttonVariants } from '@/components/ui/shadcn/button'
import { TICKET_STATUS_LABELS } from '@/domain/tickets'
import type { TicketStatus } from '@/domain/tickets/types'
import { api } from '@/repositories/api'
import { singleFlight } from '@/utils/single-flight'
import { Menu } from '@base-ui/react/menu'

import { applyStatus, statusMenuItems } from './rules'
import type { StatusActionProps } from './types'

export const STATUS_ACTION_TEXTS = {
  cancel: 'Cancelar',
  close: 'Fechar ticket',
  confirm: 'Fechar o ticket? Ninguém poderá enviar novas mensagens.',
  saving: 'Salvando…',
  trigger: 'Status'
} as const

const applyOnce = singleFlight(applyStatus)

export const StatusAction = ({ current, ticketId }: StatusActionProps) => {
  const router = useRouter()
  const [, startTransition] = useTransition()
  const [saving, setSaving] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [error, setError] = useState<null | string>(null)
  const [announcement, setAnnouncement] = useState('')
  const cancelRef = useRef<HTMLButtonElement>(null)
  const items = statusMenuItems(current)

  useEffect(() => {
    if (confirming) cancelRef.current?.focus()
  }, [confirming])

  if (items.length === 0) return null

  const refresh = () =>
    startTransition(() => {
      router.refresh()
    })

  const apply = async (to: TicketStatus) => {
    setConfirming(false)
    setError(null)
    setSaving(true)
    const outcome = await applyOnce(
      { change: api.changeStatus },
      { from: current, ticketId, to }
    )
    setSaving(false)
    if (outcome === null) return
    if (outcome.kind === 'failed') {
      setError(outcome.message)
      if (outcome.refresh) refresh()
      return
    }
    setAnnouncement(outcome.message)
    refresh()
  }

  const choose = (to: TicketStatus) => {
    if (saving || to === current) return
    if (to === 'fechado') {
      setError(null)
      setConfirming(true)
      return
    }
    void apply(to)
  }

  return (
    <div
      className="flex flex-col items-stretch gap-3 sm:items-end"
      data-testid="status-action"
    >
      <Menu.Root>
        <Menu.Trigger
          className={cn(
            buttonVariants({ variant: 'outline' }),
            'justify-between gap-2 sm:w-fit'
          )}
          aria-busy={saving || undefined}
          aria-disabled={saving || undefined}
        >
          {saving ? (
            <>
              <LoaderCircleIcon
                className="motion-safe:animate-spin"
                aria-hidden
              />
              {STATUS_ACTION_TEXTS.saving}
            </>
          ) : (
            <>
              <span>
                {STATUS_ACTION_TEXTS.trigger}:{' '}
                <span className="font-medium">
                  {TICKET_STATUS_LABELS[current]}
                </span>
              </span>
              <ChevronDownIcon className="text-muted-foreground" aria-hidden />
            </>
          )}
        </Menu.Trigger>
        <Menu.Portal>
          <Menu.Positioner align="end" className="isolate z-50" sideOffset={4}>
            <Menu.Popup className="bg-popover text-popover-foreground border-border min-w-48 origin-(--transform-origin) rounded-sm border p-1 outline-none">
              {items.map(item => (
                <Menu.Item
                  className={cn(
                    'data-highlighted:bg-muted relative flex cursor-default items-center gap-2 rounded-sm py-2 pr-8 pl-2 text-sm outline-none select-none data-disabled:pointer-events-none',
                    item.risky ? 'text-destructive' : 'text-foreground'
                  )}
                  data-status={item.status}
                  disabled={item.current}
                  key={item.status}
                  onClick={() => choose(item.status)}
                >
                  {item.label}
                  {item.current ? (
                    <CheckIcon
                      aria-label="atual"
                      className="absolute right-2 size-4"
                    />
                  ) : null}
                </Menu.Item>
              ))}
            </Menu.Popup>
          </Menu.Positioner>
        </Menu.Portal>
      </Menu.Root>

      {confirming ? (
        <div
          className="border-border bg-muted flex flex-col gap-3 rounded-sm border p-3 sm:max-w-sm"
          data-testid="close-confirmation"
          role="alert"
        >
          <p className="text-sm">{STATUS_ACTION_TEXTS.confirm}</p>
          <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <Button
              onClick={() => setConfirming(false)}
              ref={cancelRef}
              size="sm"
              type="button"
              variant="outline"
            >
              {STATUS_ACTION_TEXTS.cancel}
            </Button>
            <Button
              className="border-destructive/40 text-destructive hover:bg-destructive/5 hover:text-destructive"
              onClick={() => void apply('fechado')}
              size="sm"
              type="button"
              variant="outline"
            >
              {STATUS_ACTION_TEXTS.close}
            </Button>
          </div>
        </div>
      ) : null}

      {error ? (
        <div className="sm:max-w-sm" data-testid="status-error">
          <InlineAlert>{error}</InlineAlert>
        </div>
      ) : null}

      <p aria-live="polite" className="sr-only" data-testid="status-status">
        {announcement}
      </p>
    </div>
  )
}
