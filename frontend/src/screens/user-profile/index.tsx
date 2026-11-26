import { Forbidden } from '@/components/forbidden'
import { UserProfile } from '@/components/user-profile'
import { AppHeader } from '@/containers/app-header'
import { requireArea, requireRole } from '@/services/session'

/**
 * `/usuarios` (spec US-7): the same guard as the other role areas (the
 * page role, then the backend's own answer on GET /api/usuarios), with the
 * profile of the session user as its body. The profile data comes from
 * GET /api/me, never from the URL or the browser storage (AC-7.5).
 */
export const UserProfileScreen = async () => {
  const { allowed, user } = await requireRole('/usuarios')
  const area = allowed ? await requireArea('/api/usuarios', '/usuarios') : null

  return (
    <>
      <AppHeader role={user.role} />
      {area?.kind === 'ok' ? (
        <UserProfile
          area={area.area}
          createdAt={user.createdAt}
          login={user.login}
        />
      ) : (
        <Forbidden role={user.role} />
      )}
    </>
  )
}
