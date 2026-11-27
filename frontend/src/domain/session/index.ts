import { loginUrl } from '@/domain/navigation'
import { PROTECTED_ROUTES } from '@/domain/protected-routes'
import type { ProtectedRoute } from '@/domain/protected-routes/types'

import type { Access, Session } from './types'

export const decideAccess = (
  session: Session,
  route: ProtectedRoute,
  path: string = route
): Access => {
  switch (session.kind) {
    case 'anonymous':
      return { kind: 'login', url: loginUrl(path) }
    case 'expired':
      return { kind: 'login', url: loginUrl(path, { expired: true }) }
    case 'user':
      return session.user.role === PROTECTED_ROUTES[route]
        ? { kind: 'allowed', user: session.user }
        : { kind: 'forbidden', user: session.user }
  }
}
