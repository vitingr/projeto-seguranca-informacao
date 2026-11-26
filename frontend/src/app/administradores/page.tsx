import type { Metadata } from 'next'

import { RoleArea } from '@/screens/role-area'

export const metadata: Metadata = { title: 'Área do administrador' }

const Page = () => (
  <RoleArea
    description="Página de acesso restrito aos usuários administradores"
    endpoint="/api/administradores"
    route="/administradores"
    title="Área do administrador"
  />
)

export default Page
