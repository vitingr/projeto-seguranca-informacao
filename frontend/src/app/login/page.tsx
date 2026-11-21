import type { Metadata } from 'next'
import { redirect } from 'next/navigation'

import { LoginForm } from '@/containers/login-form'
import { safeNext } from '@/domain/navigation'
import { getSession } from '@/services/session'

export const metadata: Metadata = { title: 'Entrar' }

const Page = async ({ searchParams }: PageProps<'/login'>) => {
  const query = await searchParams
  const next = typeof query.next === 'string' ? query.next : undefined

  // A logged-in user does not see the form.
  const session = await getSession()
  if (session.kind === 'user') redirect(safeNext(next, session.user.role))

  return (
    <main className="mx-auto flex w-full max-w-sm flex-col gap-6 px-4 py-16">
      <h1 className="text-2xl font-semibold">Entrar</h1>
      <LoginForm expired={query.sessao === 'expirada'} next={next} />
    </main>
  )
}

export default Page
