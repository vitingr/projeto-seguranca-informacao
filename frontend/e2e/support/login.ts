import { type Page, expect } from '@playwright/test'

export interface Credentials {
  login: string
  password: string
}

/** Fills and submits the login form on the current page. */
export async function submitLogin(
  page: Page,
  { login, password }: Credentials
) {
  await page.getByLabel('Login').fill(login)
  await page.getByLabel('Senha').fill(password)
  await page.getByRole('button', { name: 'Entrar' }).click()
}

/** Opens /login (with an optional query) and logs in, waiting for `lands`. */
export async function loginAs(
  page: Page,
  credentials: Credentials,
  lands: RegExp | string,
  query = ''
) {
  await page.goto(`/login${query}`)
  await submitLogin(page, credentials)
  await expect(page).toHaveURL(lands)
}

/** Method and path of every /api request the page makes. */
export function recordApiCalls(page: Page) {
  const calls: string[] = []
  page.on('request', request => {
    const url = new URL(request.url())
    if (url.pathname.startsWith('/api/')) {
      calls.push(`${request.method()} ${url.pathname}`)
    }
  })
  return calls
}
