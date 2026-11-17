import { expect, test as setup } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs } from './support/login'

// Creates this run's comum user through the browser and the /api proxy, as
// the seed admin (P-4). Each run leaves one e2e.* user; the backend README
// explains how to delete them.
setup('create the comum test user', async ({ page }) => {
  await loginAs(page, seedAdmin(), '/admin')
  const comum = comumUser()
  const created = await page.evaluate(
    async ([login, senha]) => {
      const token = (await (await fetch('/api/csrf')).json()).csrf_token
      const res = await fetch('/api/cadastro', {
        body: JSON.stringify({ login, papel: 'comum', senha }),
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token },
        method: 'POST'
      })
      return { body: await res.json(), status: res.status }
    },
    [comum.login, comum.password]
  )
  expect(created.status).toBe(201)
  expect(created.body).toMatchObject({ login: comum.login, role: 'comum' })
})
