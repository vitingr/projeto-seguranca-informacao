import type { Metadata } from 'next'

import { Forbidden } from '@/components/forbidden'
import { AppHeader } from '@/containers/app-header'
import { TicketBuilder } from '@/containers/ticket-builder'
import { requireRole } from '@/services/session'

export const metadata: Metadata = { title: 'Novo ticket' }

const Page = async () => {
  const { allowed, user } = await requireRole('/usuarios/tickets/novo')
  return (
    <>
      <AppHeader role={user.role} />
      {allowed ? <TicketBuilder /> : <Forbidden role={user.role} />}
    </>
  )
}

export default Page
