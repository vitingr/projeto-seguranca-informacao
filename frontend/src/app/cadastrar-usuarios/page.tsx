import type { Metadata } from 'next'

import { requireRole } from '@/services/session'
import { AppHeader } from '@/shared/components/common/app-header'
import { Forbidden } from '@/shared/components/common/forbidden'

import { CadastroForm } from './form'

export const metadata: Metadata = { title: 'Cadastrar usuários' }

// Admin only. The backend enforces it again on POST /api/cadastro.
const Page = async () => {
  const { allowed, user } = await requireRole('/cadastrar-usuarios')
  return (
    <>
      <AppHeader role={user.role} />
      {allowed ? (
        <main className="mx-auto flex w-full max-w-sm flex-col gap-6 px-4 py-10">
          <h1 className="text-2xl font-semibold">Cadastrar usuário</h1>
          <CadastroForm />
        </main>
      ) : (
        <Forbidden role={user.role} />
      )}
    </>
  )
}

export default Page
