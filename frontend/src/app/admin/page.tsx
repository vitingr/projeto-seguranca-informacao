import type { Metadata } from 'next'

import { RoleArea } from '@/screens/role-area'

export const metadata: Metadata = { title: 'Área do administrador' }

const Page = () => (
  <RoleArea
    description="Esta área é exclusiva para administradores."
    endpoint="/api/administradores"
    route="/admin"
    title="Área do administrador"
  />
)

export default Page
