import { Forbidden } from '@/components/forbidden'
import { AppHeader } from '@/containers/app-header'
import { requireArea, requireRole } from '@/services/session'

import type { RoleAreaProps } from './types'

export const RoleArea: React.FC<RoleAreaProps> = async ({
  actions,
  description,
  endpoint,
  route,
  title
}) => {
  const { allowed, user } = await requireRole(route)
  const area = allowed ? await requireArea(endpoint, route) : null

  return (
    <>
      <AppHeader role={user.role} />
      {area?.kind === 'ok' ? (
        <main className="mx-auto flex w-full max-w-7xl flex-col gap-3 px-4 py-10 sm:px-6 lg:px-8">
          <h1 className="text-2xl font-semibold">{title}</h1>
          <p>{description}</p>
          <p className="text-muted-foreground text-sm" data-testid="area">
            Área liberada pelo servidor: {area.area}
          </p>
          {actions ? (
            <div className="flex flex-wrap gap-3 pt-3">{actions}</div>
          ) : null}
        </main>
      ) : (
        <Forbidden role={user.role} />
      )}
    </>
  )
}
