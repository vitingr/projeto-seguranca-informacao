import type { RefObject } from 'react'

export type LeaveRequest =
  | { from: HTMLAnchorElement; href: string; kind: 'link' }
  | { from: HTMLButtonElement; kind: 'logout' }

export interface GuardTarget {
  addEventListener: Window['addEventListener']
  removeEventListener: Window['removeEventListener']
}

export interface GuardOptions {
  current: () => string
  logoutAllowed: { current: boolean }
  logoutRunning: () => boolean
  onRequest: (request: LeaveRequest) => void
}

export interface LeaveGuard {
  leave: () => void
  leaving: LeaveRequest | null
  stay: () => void
  stayRef: RefObject<HTMLButtonElement | null>
}

export interface LeaveGuardBarProps {
  body: string
  leaveLabel: string
  onLeave: () => void
  onStay: () => void
  open: boolean
  stayLabel: string
  stayRef: RefObject<HTMLButtonElement | null>
  title: string
}
