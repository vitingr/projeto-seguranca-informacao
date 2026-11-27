import type { Metadata } from 'next'

import { StatusAction } from '@/containers/status-action'
import { TicketPageScreen } from '@/screens/ticket-page'

export const metadata: Metadata = { title: 'Ticket' }

// Admins only (spec 003 US-5). The backend enforces it again on every
// /api/administradores/tickets route (S-1); this page only picks the screen.
const Page = async ({
  params,
  searchParams
}: PageProps<'/administradores/tickets/[id]'>) => {
  const [{ id }, query] = await Promise.all([params, searchParams])
  return (
    <TicketPageScreen
      actions={ticket => (
        <StatusAction current={ticket.status} ticketId={ticket.id} />
      )}
      id={id}
      scope="queue"
      searchParams={query}
    />
  )
}

export default Page
