import { requireArea, requireRole } from '@/server/session'
import type { AreaPath } from '@/server/session-rules'
import { AppHeader } from '@/shared/components/app-header'
import { Forbidden } from '@/shared/components/forbidden'
import type { ProtectedRoute } from '@/shared/navigation/routes'

/**
 * A role area page: the server checks the session and the role (/api/me),
 * then loads the area from the backend, which keeps the last word (its 403
 * wins even if /api/me allowed the role a moment before).
 */
export async function RoleArea({
  description,
  endpoint,
  route,
  title
}: {
  description: string
  endpoint: AreaPath
  route: ProtectedRoute
  title: string
}) {
  const { allowed, user } = await requireRole(route)
  const area = allowed ? await requireArea(endpoint, route) : null
  return (
    <>
      <AppHeader role={user.role} />
      {area?.kind === 'ok' ? (
        <main className="mx-auto flex w-full max-w-3xl flex-col gap-3 px-4 py-10">
          <h1 className="text-2xl font-semibold">{title}</h1>
          <p>{description}</p>
          <p className="text-muted-foreground text-sm" data-testid="area">
            Área liberada pelo servidor: {area.area}
          </p>
        </main>
      ) : (
        <Forbidden role={user.role} />
      )}
    </>
  )
}
