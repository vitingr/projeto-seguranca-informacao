import type { Metadata } from 'next'

import { RoleArea } from '@/screens/role-area'
import { TicketQueue } from '@/screens/ticket-queue'

export const metadata: Metadata = { title: 'Área do administrador' }

// The queue is a child of the area, so it loads only after the role check.
const Page = async ({ searchParams }: PageProps<'/administradores'>) => (
  <RoleArea
    description="Página de acesso restrito aos usuários administradores"
    endpoint="/api/administradores"
    route="/administradores"
    title="Área do administrador"
  >
    <TicketQueue searchParams={await searchParams} />
  </RoleArea>
)

export default Page
