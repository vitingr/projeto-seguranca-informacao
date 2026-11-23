import { loginUrl } from '@/domain/navigation'
import { routeAllows } from '@/domain/protected-routes'
import type { ProtectedRoute } from '@/domain/protected-routes/types'

import type { Access, Session } from './types'

export const decideAccess = (
  session: Session,
  route: ProtectedRoute
): Access => {
  switch (session.kind) {
    case 'anonymous':
      return { kind: 'login', url: loginUrl(route) }
    case 'expired':
      return { kind: 'login', url: loginUrl(route, { expired: true }) }
    case 'user':
      return routeAllows(route, session.user.role)
        ? { kind: 'allowed', user: session.user }
        : { kind: 'forbidden', user: session.user }
  }
}
