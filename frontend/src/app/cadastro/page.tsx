import type { Metadata } from 'next'

import { Forbidden } from '@/components/forbidden'
import { AppHeader } from '@/containers/app-header'
import { CadastroForm } from '@/containers/cadastro-form'
import { requireRole } from '@/services/session'

export const metadata: Metadata = { title: 'Cadastrar usuários' }

const Page = async () => {
  const { allowed, user } = await requireRole('/cadastro')
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
