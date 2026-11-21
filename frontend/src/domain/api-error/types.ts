export interface ApiError {
  /** The backend `error` code, or `network`. */
  code: string
  /** The backend `message`, safe to show to users. */
  message?: string
  requestId?: string
  /** 0 when the request never got an answer. */
  status: number
}
