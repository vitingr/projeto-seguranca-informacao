import { type Page, expect } from '@playwright/test'

export interface Credentials {
  login: string
  password: string
}

export const submitLogin = async (
  page: Page,
  { login, password }: Credentials
) => {
  await page.getByLabel('Login').fill(login)
  await page.getByLabel('Senha').fill(password)
  await page.getByRole('button', { name: 'Entrar' }).click()
}

export const loginAs = async (
  page: Page,
  credentials: Credentials,
  lands: RegExp | string,
  query = ''
) => {
  await page.goto(`/login${query}`)
  await submitLogin(page, credentials)
  await expect(page).toHaveURL(lands)
}

export const recordApiCalls = (page: Page) => {
  const calls: string[] = []
  page.on('request', request => {
    const url = new URL(request.url())
    if (url.pathname.startsWith('/api/')) {
      calls.push(`${request.method()} ${url.pathname}`)
    }
  })
  return calls
}
