import { AppHeader } from '@/containers/app-header'
import { LIST_PATHS } from '@/domain/ticket-query'
import { TicketNotFound } from '@/screens/ticket-page'

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
