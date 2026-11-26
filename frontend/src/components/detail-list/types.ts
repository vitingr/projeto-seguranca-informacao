import type { ReactNode } from 'react'

export interface DetailListProps {
  children: ReactNode
}

export interface DetailRowProps {
  children: ReactNode
  label: string
  /** Keeps the line breaks of the value (multi-line fields). */
  preserveLines?: boolean
}
