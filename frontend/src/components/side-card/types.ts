import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

export interface SideCardProps {
  children: ReactNode
  defaultOpen?: boolean
  icon: LucideIcon
  title: string
}
