import { describe, expect, it, vi } from 'vitest'

import { parseApiUrl } from '.'

const API_BASE_URL = vi.hoisted(() => {
  const value = 'https://localhost:8443'
  vi.stubEnv('API_BASE_URL', value)
  vi.stubEnv('NODE_ENV', 'development')
  return value
})

describe('parseApiUrl', () => {
  it.each([
    [undefined, API_BASE_URL],
    ['', API_BASE_URL],
    ['  ', API_BASE_URL],
    ['https://localhost', 'https://localhost'],
    ['https://localhost/', 'https://localhost'],
    ['https://localhost:8443', 'https://localhost:8443'],
    [' https://127.0.0.1:8443 ', 'https://127.0.0.1:8443']
  ])('accepts %j', (raw, want) => {
    expect(parseApiUrl({ raw })).toBe(want)
  })

  it.each([
    ['http://localhost', /https/],
    ['localhost:8443', /https/],
    ['not a url', /absolute https URL/],
    ['https://user:secret@localhost', /credentials/],
    ['https://localhost/api', /origin/],
    ['https://localhost?x=1', /origin/],
    ['https://localhost#x', /origin/]
  ])('rejects %j', (raw, want) => {
    expect(() => parseApiUrl({ raw })).toThrow(want)
  })

  it('never repeats the value in the error', () => {
    expect(() => parseApiUrl({ raw: 'https://user:secret@localhost' })).toThrow(
      expect.objectContaining({
        message: expect.not.stringContaining('secret')
      })
    )
  })
})
