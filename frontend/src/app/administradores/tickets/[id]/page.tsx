import type { Metadata } from 'next'

import { StatusAction } from '@/containers/status-action'
import { TicketPageScreen } from '@/screens/ticket-page'

export const metadata: Metadata = { title: 'Ticket' }

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
