import type { Metadata } from 'next'

import { TicketPageScreen } from '@/screens/ticket-page'

export const metadata: Metadata = { title: 'Ticket' }

const Page = async ({
  params,
  searchParams
}: PageProps<'/usuarios/tickets/[id]'>) => {
  const [{ id }, query] = await Promise.all([params, searchParams])
  return <TicketPageScreen id={id} searchParams={query} />
}

export default Page
