import { describe, expect, it } from 'vitest'

import {
  emptyResponse,
  errorResponse,
  jsonResponse,
  textResponse
} from '@/test/fake-fetch'

import { ApiRequestError, readError, toApiError } from './errors'

describe('readError', () => {
  it('reads the error code, message and request id', async () => {
    const res = errorResponse(409, 'conflict', { request_id: 'rid-1' })
    expect(await readError(res)).toEqual({
      code: 'conflict',
      message: 'msg conflict',
      requestId: 'rid-1',
      status: 409
    })
  })

  it.each([
    ['plain text (proxy 500)', textResponse(500, 'Internal Server Error')],
    ['empty body', emptyResponse(502)],
    ['JSON without error', jsonResponse(500, { message: 'x' })],
    ['JSON with a non-string error', jsonResponse(500, { error: 1 })],
    ['JSON array', jsonResponse(500, ['internal'])],
    ['JSON null', jsonResponse(500, null)]
  ])('treats %s as a network error (D-17)', async (_name, res) => {
    expect(await readError(res)).toEqual({
      code: 'network',
      status: res.status
    })
  })

  it('omits a missing message and request id', async () => {
    const res = jsonResponse(401, { error: 'unauthorized' })
    expect(await readError(res)).toEqual({ code: 'unauthorized', status: 401 })
  })
})

describe('toApiError', () => {
  it('unwraps an ApiRequestError', () => {
    const error = { code: 'forbidden', status: 403 }
    expect(toApiError(new ApiRequestError(error))).toBe(error)
  })

  it('turns anything else into a network error', () => {
    expect(toApiError(new TypeError('fetch failed'))).toEqual({
      code: 'network',
      status: 0
    })
  })
})
