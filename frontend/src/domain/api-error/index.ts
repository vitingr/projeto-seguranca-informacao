import type { ApiError } from './types'

export const NETWORK = 'network'

export const networkError = (status = 0): ApiError => ({
  code: NETWORK,
  status
})
