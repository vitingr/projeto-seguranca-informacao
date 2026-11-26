import { type BrowserContext, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'

const sessionCookies = async (context: BrowserContext) =>
  (await context.cookies()).filter(c => c.name === '__Host-session')

test('logout ends the session, and Back shows no protected content', async ({
  context,
  page
}) => {
  await loginAs(page, seedAdmin(), '/administradores')
  await expect(page.getByTestId('area')).toBeVisible()

  // Real clicks, so Chromium keeps the history entries (spike 3). `/` only
  // redirects to the landing page now (spec AC-0.6).
  await page.getByRole('link', { name: 'Início' }).click()
  await expect(page).toHaveURL('/administradores')

  await page.getByRole('button', { name: 'Sair' }).click()
  await expect(page).toHaveURL('/login')
  expect(await sessionCookies(context)).toEqual([])
  await expectNothingInBrowserStorage(page)

  // Back to the /administradores history entry: the page is checked again on the
  // server, so it goes to login instead of showing the area.
  await page.goBack()
  await expect(page).toHaveURL('/login?next=%2Fadministradores')
  await expect(page.getByTestId('area')).toHaveCount(0)

  await page.goto('/administradores')
  await expect(page).toHaveURL('/login?next=%2Fadministradores')
})

test('logout when the session already ended still logs out', async ({
  context,
  page
}) => {
  await loginAs(page, comumUser(), '/usuarios')

  // Another tab ends the session first.
  const other = await context.newPage()
  await other.goto('/')
  const status = await other.evaluate(async () => {
    const token = (await (await fetch('/api/csrf')).json()).csrf_token
    return (
      await fetch('/api/logout', {
        headers: { 'X-CSRF-Token': token },
        method: 'POST'
      })
    ).status
  })
  expect(status).toBe(204)

  // This tab still shows the area; its logout gets 403 (no session), then
  // retries with a new token and gets 401, which counts as logged out.
  await page.getByRole('button', { name: 'Sair' }).click()
  await expect(page).toHaveURL('/login')
  await expect(page.getByTestId('form-error')).toHaveCount(0)
})
