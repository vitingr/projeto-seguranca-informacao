import { requireArea, requireRole } from '@/server/session'
import { AppHeader } from '@/shared/components/common/app-header'
import { Forbidden } from '@/shared/components/common/forbidden'

import type { RoleAreaProps } from './types'

export const RoleArea: React.FC<RoleAreaProps> = async ({
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
