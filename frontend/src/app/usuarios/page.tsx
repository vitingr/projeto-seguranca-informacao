import type { Metadata } from 'next'
import Link from 'next/link'

import { buttonVariants } from '@/components/ui/shadcn/button'
import { RoleArea } from '@/screens/role-area'

export const metadata: Metadata = { title: 'Minha conta' }

const Page = () => (
  <RoleArea
    actions={
      <Link
        className={buttonVariants({ size: 'lg' })}
        href="/usuarios/tickets/novo"
      >
        Novo ticket
      </Link>
    }
    description="Página de acesso restrito aos usuários comuns"
    endpoint="/api/usuarios"
    route="/usuarios"
    title="Minha conta"
  />
)

export default Page
