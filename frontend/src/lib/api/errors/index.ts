import { networkError } from '@/domain/api-error'
import type { ApiError } from '@/domain/api-error/types'

export class ApiRequestError extends Error {
  constructor(readonly error: ApiError) {
    super(`api request failed: ${error.status} ${error.code}`)
    this.name = 'ApiRequestError'
  }
}

const optionalString = (value: unknown) =>
  typeof value === 'string' && value !== '' ? value : undefined

export const readError = async (res: Response): Promise<ApiError> => {
  let body: unknown
  try {
    body = await res.json()
  } catch {
    return networkError(res.status)
  }
  if (typeof body !== 'object' || body === null) {
    return networkError(res.status)
  }
  const fields = body as Record<string, unknown>
  const code = optionalString(fields.error)
  if (!code) return networkError(res.status)
  return {
    code,
    message: optionalString(fields.message),
    requestId: optionalString(fields.request_id),
    status: res.status
  }
}

export const toApiError = (err: unknown): ApiError => {
  return err instanceof ApiRequestError ? err.error : networkError()
}
