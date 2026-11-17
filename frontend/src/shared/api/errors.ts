// Errors from the backend, read from the `error` field of ErrorResponse
// (spec D-8). Anything that is not a JSON ErrorResponse (backend down or
// untrusted: the proxy answers a plain-text 500, or fetch itself fails) is
// reported as `network` (spec D-17).

export const NETWORK = 'network'

export interface ApiError {
  /** The backend `error` code, or `network`. */
  code: string
  /** The backend `message`, safe to show to users. */
  message?: string
  requestId?: string
  /** 0 when the request never got an answer. */
  status: number
}

export class ApiRequestError extends Error {
  constructor(readonly error: ApiError) {
    super(`api request failed: ${error.status} ${error.code}`)
    this.name = 'ApiRequestError'
  }
}

export const networkError = (status = 0): ApiError => ({
  code: NETWORK,
  status
})

const optionalString = (value: unknown) =>
  typeof value === 'string' && value !== '' ? value : undefined

/** Reads an error response. It never throws. */
export async function readError(res: Response): Promise<ApiError> {
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

/** Turns anything thrown by the API layer into an ApiError. */
export function toApiError(err: unknown): ApiError {
  return err instanceof ApiRequestError ? err.error : networkError()
}
