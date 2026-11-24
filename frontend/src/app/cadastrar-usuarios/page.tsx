import type { Metadata } from 'next'

import { Forbidden } from '@/components/forbidden'
import { AppHeader } from '@/containers/app-header'
import { CadastroScreen } from '@/screens/cadastro'
import { requireRole } from '@/services/session'

export const metadata: Metadata = { title: 'Cadastrar usuários' }

// Admin only. The backend enforces it again on POST /api/cadastro.
const Page = async () => {
  const { allowed, user } = await requireRole('/cadastrar-usuarios')
  return (
    <>
      <AppHeader role={user.role} />
      {allowed ? <CadastroScreen /> : <Forbidden role={user.role} />}
    </>
  )
}

export default Page
