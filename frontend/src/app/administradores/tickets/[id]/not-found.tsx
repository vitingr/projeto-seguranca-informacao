import { AppHeader } from '@/containers/app-header'
import { LIST_PATHS } from '@/domain/ticket-query'
import { TicketNotFound } from '@/screens/ticket-page'

// Rendered by `notFound()` for a missing ticket or a malformed number (spec
// 003 AC-5.2). Only an admin gets here: the page checks the role first.
const NotFound = () => (
  <>
    <AppHeader role="admin" />
    <TicketNotFound
      backHref={LIST_PATHS.queue}
      backLabel="Voltar para a fila"
    />
  </>
)

export default NotFound
