'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useRef, useState } from 'react'

import { Button } from '@/components/ui/shadcn/button'
import { LOGOUT_SELECTOR } from '@/containers/logout-button'
import { isPlainClick, leavesPage } from '@/utils/navigation-guard'

import type {
  GuardOptions,
  GuardTarget,
  LeaveGuard,
  LeaveGuardBarProps,
  LeaveRequest
} from './types'

// The unsent data guard (001 AC-3.8, 003 AC-3.9, plan P-12), shared by the
// ticket builder and the reply box: with typed text not sent, closing or
// reloading the tab gets the browser's own prompt, and a link out of the
// page or the header's "Sair" get an inline bar instead of a dialog.

/**
 * Listens for leaving on `target` (the window) and returns the cleanup.
 * `beforeunload` asks the browser to warn, unless a confirmed logout is
 * running. A capture-phase click listener runs before the router's link
 * handler and the button's own handler: it stops a click on "Sair" or on a
 * plain link out of the page and reports it through `onRequest`.
 */
export const installLeaveGuard = (
  target: GuardTarget,
  options: GuardOptions
): (() => void) => {
  const onBeforeUnload = (event: Event) => {
    // A logout the user confirmed is running (the button stays disabled
    // until the login page loads): its redirect is not warned again. If the
    // logout fails, the button is enabled and the warning is back.
    if (options.logoutRunning()) return
    event.preventDefault()
  }
  const onClick = (event: Event) => {
    if (!(event.target instanceof Element)) return
    const logout = event.target.closest(LOGOUT_SELECTOR)
    if (logout instanceof HTMLButtonElement) {
      if (options.logoutAllowed.current) {
        options.logoutAllowed.current = false
        return
      }
      // Any click on "Sair" logs out, with or without a modifier.
      event.preventDefault()
      event.stopPropagation()
      options.onRequest({ from: logout, kind: 'logout' })
      return
    }
    if (!isPlainClick(event as MouseEvent)) return
    const link = event.target.closest('a[href]')
    if (
      !(link instanceof HTMLAnchorElement) ||
      (link.target !== '' && link.target !== '_self') ||
      link.hasAttribute('download') ||
      !leavesPage(link.href, options.current())
    ) {
      return
    }
    event.preventDefault()
    event.stopPropagation()
    options.onRequest({ from: link, href: link.href, kind: 'link' })
  }
  target.addEventListener('beforeunload', onBeforeUnload)
  target.addEventListener('click', onClick, true)
  return () => {
    target.removeEventListener('beforeunload', onBeforeUnload)
    target.removeEventListener('click', onClick, true)
  }
}

/** The guard of a page: active only while `active` (typed and not sent). */
export const useLeaveGuard = (active: boolean): LeaveGuard => {
  const router = useRouter()
  const [leaving, setLeaving] = useState<LeaveRequest | null>(null)
  const stayRef = useRef<HTMLButtonElement>(null)
  const logoutAllowed = useRef(false)

  useEffect(() => {
    if (!active) return
    return installLeaveGuard(window, {
      current: () => window.location.href,
      logoutAllowed,
      logoutRunning: () =>
        document.querySelector(`${LOGOUT_SELECTOR}:disabled`) !== null,
      onRequest: setLeaving
    })
  }, [active])

  useEffect(() => {
    if (leaving) stayRef.current?.focus()
  }, [leaving])

  const stay = () => {
    leaving?.from.focus()
    setLeaving(null)
  }

  const leave = () => {
    if (!leaving) return
    setLeaving(null)
    if (leaving.kind === 'logout') {
      // The button logs out as it always does (POST /api/logout, then the
      // login page), showing its own error if that fails.
      logoutAllowed.current = true
      leaving.from.click()
      logoutAllowed.current = false
      return
    }
    const url = new URL(leaving.href)
    router.push(`${url.pathname}${url.search}${url.hash}`)
  }

  return { leave, leaving, stay, stayRef }
}

/** The inline "Sair sem enviar?" bar, shown while a request is pending. */
export const LeaveGuardBar = ({
  body,
  leaveLabel,
  onLeave,
  onStay,
  open,
  stayLabel,
  stayRef,
  title
}: LeaveGuardBarProps) =>
  open ? (
    <div
      className="border-border bg-muted flex flex-col gap-3 rounded-sm border p-4 sm:flex-row sm:items-center sm:justify-between"
      data-testid="leave-guard"
      role="alert"
    >
      <p className="text-sm">
        <span className="font-medium">{title}</span> {body}
      </p>
      <div className="flex shrink-0 flex-col-reverse gap-2 sm:flex-row">
        <Button
          className="w-full sm:w-auto"
          onClick={onStay}
          ref={stayRef}
          size="sm"
          type="button"
          variant="outline"
        >
          {stayLabel}
        </Button>
        <Button
          className="w-full sm:w-auto"
          onClick={onLeave}
          size="sm"
          type="button"
        >
          {leaveLabel}
        </Button>
      </div>
    </div>
  ) : null
