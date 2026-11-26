import type { ReactNode } from 'react'

import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { AreaPath } from '@/domain/session/types'

export interface RoleAreaProps {
  /** Actions shown under the description, such as "Novo ticket". */
  actions?: ReactNode
  description: string
  endpoint: AreaPath
  route: ProtectedRoute
  title: string
}
