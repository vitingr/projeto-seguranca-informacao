import { randomBytes } from 'node:crypto'
import { describe, expect, it } from 'vitest'

import { generateComumCredentials } from './env'

describe('generateComumCredentials', () => {
  it('follows the backend login rule and password policy', () => {
    const { login, password } = generateComumCredentials(randomBytes)
    expect(login).toMatch(/^e2e\.[0-9a-f]{24}$/)
    expect(login).toMatch(/^[a-z0-9._-]{3,64}$/)
    expect(password.length).toBeGreaterThanOrEqual(15)
    expect(password.length).toBeLessThanOrEqual(128)
  })

  it('is different on every call', () => {
    const a = generateComumCredentials(randomBytes)
    const b = generateComumCredentials(randomBytes)
    expect(a.login).not.toBe(b.login)
    expect(a.password).not.toBe(b.password)
  })
})
