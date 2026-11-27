export interface BackendAnswer {
  body: unknown
  status: number
}

export type BackendGet = (
  path: string,
  session: string
) => Promise<BackendAnswer>
