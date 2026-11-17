import Link from 'next/link'

import { getSession } from '@/server/session'
import { AppHeader } from '@/shared/components/common/app-header'
import { buttonVariants } from '@/shared/components/ui/shadcn/button'
import { ROLE_LINKS } from '@/shared/constants/roles/role-links'

// Public page (spec D-10). Without a session cookie it does not call the
// API; with one, it shows the links of the user's role.
const Page = async () => {
  const session = await getSession()

  if (session.kind !== 'user') {
    return (
      <main className="mx-auto flex w-full max-w-md flex-col gap-4 px-4 py-16">
        <h1 className="text-2xl font-semibold">Sistemas da Informação</h1>
        <p>Entre para acessar as áreas do seu papel.</p>
        <Link className={buttonVariants({ size: 'lg' })} href="/login">
          Entrar
        </Link>
      </main>
    )
  }

  const { role } = session.user
  return (
    <>
      <AppHeader role={role} />
      <main className="mx-auto flex w-full max-w-3xl flex-col gap-4 px-4 py-10">
        <h1 className="text-2xl font-semibold">Início</h1>
        <ul className="flex flex-col gap-2" data-testid="home-links">
          {ROLE_LINKS[role].map((link, index: number) => (
            <li key={`${link.href}-${index}`}>
              <Link className="hover:underline" href={link.href}>
                {link.label}
              </Link>
            </li>
          ))}
        </ul>
      </main>
    </>
  )
}

export default Page
