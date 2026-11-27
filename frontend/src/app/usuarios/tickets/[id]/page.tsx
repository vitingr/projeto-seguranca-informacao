import type { Metadata } from 'next'

import { TicketPageScreen } from '@/screens/ticket-page'

// The number is not in the title: the title stays the same for a ticket
// that is not the user's (S-2).
export const metadata: Metadata = { title: 'Ticket' }

const Page = async ({
  params,
  searchParams
}: PageProps<'/usuarios/tickets/[id]'>) => {
  const [{ id }, query] = await Promise.all([params, searchParams])
  return <TicketPageScreen id={id} searchParams={query} />
}

export default Page
