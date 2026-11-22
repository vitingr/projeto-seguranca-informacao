export interface RecordedCall {
  body?: unknown
  headers: Record<string, string>
  method: string
  url: string
}

export type Reply = Error | Response | ((call: RecordedCall) => Response)

export const jsonResponse = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
    status
  })

export const errorResponse = (
  status: number,
  error: string,
  extra: Record<string, string> = {}
) => jsonResponse(status, { error, message: `msg ${error}`, ...extra })

export const emptyResponse = (status: number) => new Response(null, { status })

export const textResponse = (status: number, text: string) =>
  new Response(text, { status })

let tokenCounter = 0
export const csrfReply = () =>
  jsonResponse(200, {
    csrf_token: `token-${++tokenCounter}`,
    header: 'X-CSRF-Token'
  })

export const fakeFetch = (routes: Record<string, Reply[]>) => {
  const calls: RecordedCall[] = []
  const fetchFn: typeof fetch = async (input, init) => {
    const url = String(input)
    const headers = Object.fromEntries(new Headers(init?.headers).entries())
    const call: RecordedCall = {
      body: typeof init?.body === 'string' ? JSON.parse(init.body) : undefined,
      headers,
      method: init?.method ?? 'GET',
      url
    }
    calls.push(call)
    const key = `${call.method} ${url}`
    const reply = routes[key]?.shift()
    if (!reply) throw new Error(`unexpected request ${key}`)
    if (reply instanceof Error) throw reply
    return typeof reply === 'function' ? reply(call) : reply
  }
  return { calls, fetch: fetchFn }
}

export const settled = async <T>(promise: Promise<T>) => {
  const marker = Symbol('pending')
  const result = await Promise.race([
    promise,
    new Promise<typeof marker>(resolve => setTimeout(() => resolve(marker), 20))
  ])
  return result === marker ? 'pending' : result
}
