import type { Metadata } from 'next'

import { RoleArea } from '@/shared/components/role-area'

export const metadata: Metadata = { title: 'Minha conta' }

const Page = () => (
  <RoleArea
    description="Esta área é exclusiva para usuários comuns."
    endpoint="/api/usuarios"
    route="/minha-conta"
    title="Minha conta"
  />
)

export default Page
