import { randomBytes } from 'node:crypto'

import { type Browser, type Locator, type Page, expect } from '@playwright/test'

import { generateComumCredentials, seedAdmin } from './env'
import { type Credentials, loginAs } from './login'

// Helpers of the post wall tests (specs 005 and 006). Every call goes
// through the browser and the /api proxy, with the session cookie and the
// CSRF token of the page.

export const unique = () => randomBytes(6).toString('hex')

export interface ApiAnswer {
  body: unknown
  status: number
}

/** A call from the page, with the CSRF token, as a script would do it. */
export const callApi = (
  page: Page,
  method: string,
  path: string,
  body?: unknown
): Promise<ApiAnswer> =>
  page.evaluate(
    async ([m, p, b]) => {
      const token = (await (await fetch('/api/csrf')).json()).csrf_token
      const headers: Record<string, string> = { 'X-CSRF-Token': token }
      if (b !== undefined) headers['Content-Type'] = 'application/json'
      const res = await fetch(p, {
        body: b === undefined ? undefined : JSON.stringify(b),
        headers,
        method: m
      })
      const text = await res.text()
      return { body: text ? JSON.parse(text) : null, status: res.status }
    },
    [method, path, body] as const
  )

/** A new comum user, registered by the seed admin. */
export const registerComum = async (browser: Browser): Promise<Credentials> => {
  const user = generateComumCredentials(randomBytes)
  const page = await browser.newPage()
  await loginAs(page, seedAdmin(), '/admin')
  const created = await callApi(page, 'POST', '/api/cadastro', {
    login: user.login,
    papel: 'comum',
    senha: user.password
  })
  expect(created.status).toBe(201)
  await page.close()
  return user
}

/** Logs in on a new page of its own (its own cookie jar). */
export const pageAs = async (
  browser: Browser,
  user: Credentials,
  lands = '/minha-conta'
) => {
  const context = await browser.newContext()
  const page = await context.newPage()
  await loginAs(page, user, lands)
  return page
}

export const createViaApi = async (
  page: Page,
  title: string,
  body = 'Corpo.'
) => {
  const created = await callApi(page, 'POST', '/api/posts', { body, title })
  expect(created.status).toBe(201)
  return (created.body as { id: number }).id
}

export const postNamed = (page: Page, title: string): Locator =>
  page
    .getByTestId('post')
    .filter({ has: page.getByRole('heading', { name: title, exact: true }) })
