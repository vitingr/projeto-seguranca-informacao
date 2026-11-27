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

  await page.getByRole('link', { name: 'Início' }).click()
  await expect(page).toHaveURL('/administradores')

  await page.getByRole('button', { name: 'Sair' }).click()
  await expect(page).toHaveURL('/login')
  expect(await sessionCookies(context)).toEqual([])
  await expectNothingInBrowserStorage(page)

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

  await page.getByRole('button', { name: 'Sair' }).click()
  await expect(page).toHaveURL('/login')
  await expect(page.getByTestId('form-error')).toHaveCount(0)
})
