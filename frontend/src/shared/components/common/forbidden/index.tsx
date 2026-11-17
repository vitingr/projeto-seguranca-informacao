import Link from 'next/link'

import { MESSAGES } from '@/shared/api/messages'
import { buttonVariants } from '@/shared/components/ui/shadcn/button'
import { landingFor } from '@/shared/utils/navigation'

import type { ForbiddenProps } from './types'

export const Forbidden: React.FC<ForbiddenProps> = ({ role }) => (
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
