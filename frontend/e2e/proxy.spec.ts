import { expect, test } from '@playwright/test'

import { seedAdmin } from './support/env'
import { expectNothingInBrowserStorage } from './support/storage'

const ORIGIN = 'https://localhost:3000'

interface Answer {
  body: unknown
  status: number
}

test('the browser reaches the API only through the frontend origin', async ({
  context,
  page
}) => {
  const requested: string[] = []
  page.on('request', request => requested.push(request.url()))

  await page.goto('/')

  const csrf = await page.evaluate<Answer>(async () => {
    const res = await fetch('/api/csrf')
    return { body: await res.json(), status: res.status }
  })
  expect(csrf.status).toBe(200)
  const anonymousToken = (csrf.body as { csrf_token: string }).csrf_token

  // The cookie keeps the attributes the backend set (spec: HTTPS and cookie).
  const [anonymous] = await context.cookies()
  expect(anonymous).toMatchObject({
    domain: 'localhost',
    expires: -1,
    httpOnly: true,
    name: '__Host-session',
    path: '/',
    sameSite: 'Lax',
    secure: true
  })

  // Without the token the backend refuses the POST, through the proxy too.
  const noToken = await page.evaluate(async () => {
    const res = await fetch('/api/login', {
      body: JSON.stringify({ login: 'x', senha: 'y' }),
      headers: { 'Content-Type': 'application/json' },
      method: 'POST'
    })
    return res.status
  })
  expect(noToken).toBe(403)

  // A browser POST passes the Fetch Metadata check (Sec-Fetch-Site:
  // same-origin) and the token check.
  const admin = seedAdmin()
  const login = await page.evaluate<Answer, [string, string, string]>(
    async ([token, user, password]) => {
      const res = await fetch('/api/login', {
        body: JSON.stringify({ login: user, senha: password }),
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token },
        method: 'POST'
      })
      return { body: await res.json(), status: res.status }
    },
    [anonymousToken, admin.login, admin.password]
  )
  expect(login.status).toBe(200)
  expect(login.body).toMatchObject({ role: 'admin' })

  const [authenticated] = await context.cookies()
  expect(authenticated?.name).toBe('__Host-session')
  expect(authenticated?.value).not.toBe(anonymous?.value)

  // The token changes at login (002 D-4).
  const loggedInToken = await page.evaluate(
    async () => (await (await fetch('/api/csrf')).json()).csrf_token as string
  )
  expect(loggedInToken).not.toBe(anonymousToken)

  const logout = await page.evaluate(async token => {
    const res = await fetch('/api/logout', {
      headers: { 'X-CSRF-Token': token },
      method: 'POST'
    })
    return res.status
  }, loggedInToken)
  expect(logout).toBe(204)
  expect(await context.cookies()).toEqual([])

  expect(requested.length).toBeGreaterThan(0)
  for (const url of requested) {
    expect(new URL(url).origin).toBe(ORIGIN)
  }
  await expectNothingInBrowserStorage(page)
})
