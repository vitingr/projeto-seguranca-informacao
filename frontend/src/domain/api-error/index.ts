import type { ApiError } from './types'

export const NETWORK = 'network'

export const TICKET_CLOSED = 'ticket_closed'

export const STALE_STATUS = 'stale_status'

export const networkError = (status = 0): ApiError => ({
  code: NETWORK,
  status
})
