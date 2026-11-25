import type { Metadata } from 'next'

import { RoleArea } from '@/screens/role-area'

export const metadata: Metadata = { title: 'Minha conta' }

const Page = () => (
  <RoleArea
    description="Página de acesso restrito aos usuários comuns"
    endpoint="/api/usuarios"
    route="/usuarios"
    title="Minha conta"
  />
)

export default Page
