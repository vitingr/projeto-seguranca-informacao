import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

export interface UserProfileProps {
  area: string
  children?: ReactNode
  createdAt?: string
  login?: string
}

export interface ProfileCardProps {
  children: ReactNode
  icon: LucideIcon
  id: string
  title: string
}
