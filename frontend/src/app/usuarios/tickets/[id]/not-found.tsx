import { AppHeader } from '@/containers/app-header'
import { LIST_PATHS } from '@/domain/ticket-query'
import { TicketNotFound } from '@/screens/ticket-page'

// Rendered by `notFound()` for a ticket of someone else, a missing ticket or
// a malformed number, always the same (spec 003 AC-2.4, S-2). Only a comum
// user gets here: the page checks the role before loading the ticket.
const NotFound = () => (
  <>
    <AppHeader role="comum" />
    <TicketNotFound
      backHref={LIST_PATHS.own}
      backLabel="Voltar para meus tickets"
    />
  </>
)

export default NotFound
