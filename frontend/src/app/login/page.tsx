import type { Metadata } from 'next'
import { redirect } from 'next/navigation'

import { safeNext } from '@/domain/navigation'
import { LoginScreen } from '@/screens/login'
import { getSession } from '@/services/session'

export const metadata: Metadata = { title: 'Entrar' }

const Page = async ({ searchParams }: PageProps<'/login'>) => {
  const query = await searchParams
  const next = typeof query.next === 'string' ? query.next : undefined

  // A logged-in user does not see the form.
  const session = await getSession()
  if (session.kind === 'user') redirect(safeNext(next, session.user.role))

  return <LoginScreen expired={query.sessao === 'expirada'} next={next} />
}

export default Page
