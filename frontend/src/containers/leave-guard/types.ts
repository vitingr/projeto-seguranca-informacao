import type { RefObject } from 'react'

/** Where the user tried to go while something typed was not sent. */
export type LeaveRequest =
  | { from: HTMLAnchorElement; href: string; kind: 'link' }
  | { from: HTMLButtonElement; kind: 'logout' }

/** The parts of `window` and `document` the guard listens on. */
export interface GuardTarget {
  addEventListener: Window['addEventListener']
  removeEventListener: Window['removeEventListener']
}

export interface GuardOptions {
  /** The page's current address, read at click time. */
  current: () => string
  /** The one click on "Sair" let through after "Sair sem enviar". */
  logoutAllowed: { current: boolean }
  /** Whether a confirmed logout is already running (its button disabled). */
  logoutRunning: () => boolean
  onRequest: (request: LeaveRequest) => void
}

export interface LeaveGuard {
  /** Goes where the user wanted to go: the link, or the logout. */
  leave: () => void
  /** The pending request, while the inline bar is shown. */
  leaving: LeaveRequest | null
  /** Closes the bar and gives the focus back to what was clicked. */
  stay: () => void
  /** The "stay" button, focused when the bar opens. */
  stayRef: RefObject<HTMLButtonElement | null>
}

export interface LeaveGuardBarProps {
  body: string
  leaveLabel: string
  onLeave: () => void
  onStay: () => void
  /** Shown only while a request is pending. */
  open: boolean
  stayLabel: string
  stayRef: RefObject<HTMLButtonElement | null>
  title: string
}
