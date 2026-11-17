import { describe, expect, it } from 'vitest'

import { DEFAULT_API_URL, parseApiUrl } from './api-url'

describe('parseApiUrl', () => {
  it.each([
    [undefined, DEFAULT_API_URL],
    ['', DEFAULT_API_URL],
    ['  ', DEFAULT_API_URL],
    ['https://localhost', 'https://localhost'],
    ['https://localhost/', 'https://localhost'],
    ['https://localhost:8443', 'https://localhost:8443'],
    [' https://127.0.0.1:8443 ', 'https://127.0.0.1:8443']
  ])('accepts %j', (raw, want) => {
    expect(parseApiUrl(raw)).toBe(want)
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
    expect(() => parseApiUrl(raw)).toThrow(want)
  })

  it('never repeats the value in the error', () => {
    expect(() => parseApiUrl('https://user:secret@localhost')).toThrow(
      expect.objectContaining({
        message: expect.not.stringContaining('secret')
      })
    )
  })
})
