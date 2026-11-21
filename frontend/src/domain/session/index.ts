import { loginUrl } from '@/domain/navigation'
import { PROTECTED_ROUTES } from '@/domain/protected-routes'
import type { ProtectedRoute } from '@/domain/protected-routes/types'

import type { Access, Session } from './types'

export function decideAccess(session: Session, route: ProtectedRoute): Access {
  switch (session.kind) {
    case 'anonymous':
      return { kind: 'login', url: loginUrl(route) }
    case 'expired':
      return { kind: 'login', url: loginUrl(route, { expired: true }) }
    case 'user':
      return session.user.role === PROTECTED_ROUTES[route]
        ? { kind: 'allowed', user: session.user }
        : { kind: 'forbidden', user: session.user }
  }
}
