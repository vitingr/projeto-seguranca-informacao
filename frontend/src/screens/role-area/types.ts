import type { ReactNode } from 'react'

import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { AreaPath } from '@/domain/session/types'

export interface RoleAreaProps {
  children?: ReactNode
  description: string
  endpoint: AreaPath
  route: ProtectedRoute
  title: string
}
