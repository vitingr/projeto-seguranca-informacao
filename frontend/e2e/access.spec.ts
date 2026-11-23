import { type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs, recordApiCalls } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'

const FORBIDDEN = 'Você não tem permissão para acessar esta página.'

const expectArea = async (page: Page, path: string, area: string) => {
  await page.goto(path)
  await expect(page).toHaveURL(path)
  await expect(page.getByTestId('area')).toContainText(area)
  await expect(page.getByTestId('forbidden')).toHaveCount(0)
}

const expectForbidden = async (page: Page, path: string) => {
  await page.goto(path)
  await expect(page).toHaveURL(path)
  await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
  await expect(page.getByTestId('area')).toHaveCount(0)
}

test.describe('anonymous', () => {
  test('home is public and offers the login', async ({ page }) => {
    const calls = recordApiCalls(page)
    await page.goto('/')
    await expect(page.getByRole('link', { name: 'Entrar' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sair' })).toHaveCount(0)
    expect(calls).toEqual([])
  })

  for (const path of ['/admin', '/minha-conta']) {
    test(`${path} goes to login`, async ({ page }) => {
      await page.goto(path)
      await expect(page).toHaveURL(`/login?next=${encodeURIComponent(path)}`)
    })
  }
})

test.describe('comum', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/minha-conta')
  })

  test('sees its area, and the admin area is forbidden', async ({ page }) => {
    await expectArea(page, '/minha-conta', 'usuarios')
    await expectForbidden(page, '/admin')
    await expectNothingInBrowserStorage(page)
  })

  test('home shows only the comum links', async ({ page }) => {
    await page.goto('/')
    const links = page.getByTestId('home-links')
    await expect(links.getByRole('link')).toHaveText(['Minha conta', 'Mural'])
    await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
  })

  test('/login sends it to its landing page', async ({ page }) => {
    await page.goto('/login')
    await expect(page).toHaveURL('/minha-conta')
  })
})

test.describe('admin', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, seedAdmin(), '/admin')
  })

  test('sees its area, and the comum area is forbidden', async ({ page }) => {
    await expectArea(page, '/admin', 'administradores')
    await expectForbidden(page, '/minha-conta')
  })

  test('home shows the admin links', async ({ page }) => {
    await page.goto('/')
    const links = page.getByTestId('home-links')
    await expect(links.getByRole('link')).toHaveText([
      'Área do administrador',
      'Cadastrar usuários',
      'Mural',
      'Auditoria'
    ])
  })

  test('the forbidden screen links back to the landing page', async ({
    page
  }) => {
    await page.goto('/minha-conta')
    await page
      .getByRole('link', { name: 'Ir para a minha página inicial' })
      .click()
    await expect(page).toHaveURL('/admin')
  })
})
