import type { ReactNode } from 'react'

export interface ThreadMessageProps {
  /** The mark in front of a message of the other side; none for own ones. */
  mark?: ReactNode
  /** "Você", "Suporte" or a login. */
  name: string
  /** The server's time of the page render, for the short time format. */
  now: string
  /** Written by the viewer: on the right, on a gray card, with no mark. */
  own: boolean
  text: string
  /** When the message was stored (ISO 8601). */
  time: string
}

export interface ThreadTimeProps {
  iso: string
  now: string
}
