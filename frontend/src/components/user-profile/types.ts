import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

export interface UserProfileProps {
  /** What GET /api/usuarios answered ("usuarios"). */
  area: string
  /** Sections under the profile ("Meus tickets", spec 003 AC-1.1). */
  children?: ReactNode
  /** The registration date from GET /api/me (ISO 8601). */
  createdAt?: string
  /** The login from GET /api/me. */
  login?: string
}

export interface ProfileCardProps {
  children: ReactNode
  icon: LucideIcon
  /** The id of the title, which names the card. */
  id: string
  title: string
}
