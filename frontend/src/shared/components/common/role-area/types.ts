import type { AreaPath } from '@/server/session-rules'
import type { ProtectedRoute } from '@/shared/types/navigation/protected-route'

export interface RoleAreaProps {
  description: string
  endpoint: AreaPath
  route: ProtectedRoute
  title: string
}
