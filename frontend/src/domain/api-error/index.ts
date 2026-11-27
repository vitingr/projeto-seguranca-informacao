import type { ApiError } from './types'

export const NETWORK = 'network'

/** A reply to a "Fechado" ticket (409 of a message, spec 003 AC-3.7). */
export const TICKET_CLOSED = 'ticket_closed'

/** A status changed meanwhile (409 of a status change, spec 003 AC-6.3). */
export const STALE_STATUS = 'stale_status'

export const networkError = (status = 0): ApiError => ({
  code: NETWORK,
  status
})
