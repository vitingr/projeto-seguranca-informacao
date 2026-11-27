import { AppHeader } from '@/containers/app-header'
import { LIST_PATHS } from '@/domain/ticket-query'
import { TicketNotFound } from '@/screens/ticket-page'

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
