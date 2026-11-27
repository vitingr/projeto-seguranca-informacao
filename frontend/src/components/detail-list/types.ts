import type { ReactNode } from 'react'

export interface DetailListProps {
  children: ReactNode
}

export interface DetailRowProps {
  children: ReactNode
  label: string
  preserveLines?: boolean
}
