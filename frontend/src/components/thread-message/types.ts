import type { ReactNode } from 'react'

export interface ThreadMessageProps {
  entryId?: number
  mark?: ReactNode
  name: string
  now: string
  own: boolean
  text: string
  time: string
}

export interface ThreadTimeProps {
  iso: string
  now: string
}
