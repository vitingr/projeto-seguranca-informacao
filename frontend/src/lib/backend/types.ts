export interface BackendAnswer {
  /** Parsed JSON, or null when the body is not JSON. */
  body: unknown
  status: number
}

export type BackendGet = (
  path: string,
  session: string
) => Promise<BackendAnswer>
