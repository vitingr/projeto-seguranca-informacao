import type { Metadata } from 'next'

import { Forbidden } from '@/components/forbidden'
import { AppHeader } from '@/containers/app-header'
import { TicketBuilder } from '@/containers/ticket-builder'
import { requireRole } from '@/services/session'

export const metadata: Metadata = { title: 'Novo ticket' }

// Comum users only. The backend enforces it again on POST /api/tickets
// (spec S-6); this page only picks the screen.
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
