import Link from 'next/link'

import { buttonVariants } from '@/components/ui/shadcn/button'
import { MESSAGES } from '@/domain/messages'
import { landingFor } from '@/domain/navigation'

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
