import Link from 'next/link'

import { MESSAGES } from '@/shared/api/messages'
import type { Role } from '@/shared/api/types'
import { buttonVariants } from '@/shared/components/ui/shadcn/button'
import { landingFor } from '@/shared/navigation/routes'

/**
 * The "no permission" screen for a logged-in user on another role's page.
 * A normal page (HTTP 200), with no experimental Next flag (spec D-13).
 */
export const Forbidden = ({ role }: { role: Role }) => (
  <main className="mx-auto flex w-full max-w-md flex-col gap-4 px-4 py-16">
    <h1 className="text-xl font-semibold">Acesso negado</h1>
    <p data-testid="forbidden" role="alert">
      {MESSAGES.forbiddenPage}
    </p>
    <Link
      className={buttonVariants({ variant: 'outline' })}
      href={landingFor(role)}
    >
      Ir para a minha página inicial
    </Link>
  </main>
)
