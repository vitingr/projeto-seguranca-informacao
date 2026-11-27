import { type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs, recordApiCalls } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'

const FORBIDDEN = 'Você não tem permissão para acessar esta página.'
const COMUM_TEXT = 'Página de acesso restrito aos usuários comuns'
const ADMIN_TEXT = 'Página de acesso restrito aos usuários administradores'

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

const headerLinks = (page: Page) =>
  page.getByRole('navigation', { name: 'Principal' }).getByRole('link')

test.describe('anonymous', () => {
  test('/ goes to the login page', async ({ page }) => {
    const calls = recordApiCalls(page)
    await page.goto('/')
    await expect(page).toHaveURL('/login')
    await expect(page.getByRole('heading', { name: 'Entrar' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sair' })).toHaveCount(0)
    expect(calls).toEqual([])
  })

  for (const path of ['/administradores', '/usuarios']) {
    test(`${path} goes to login`, async ({ page }) => {
      await page.goto(path)
      await expect(page).toHaveURL(`/login?next=${encodeURIComponent(path)}`)
    })
  }
})

test.describe('comum', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
  })

  test('sees its area, and the admin area is forbidden', async ({ page }) => {
    await expectArea(page, '/usuarios', 'usuarios')
    await expect(page.getByText(COMUM_TEXT, { exact: true })).toBeVisible()
    await expectForbidden(page, '/administradores')
    await expectNothingInBrowserStorage(page)
  })

  test('/ goes to its area, and the header shows only the comum links', async ({
    page
  }) => {
    await page.goto('/')
    await expect(page).toHaveURL('/usuarios')
    await expect(headerLinks(page)).toHaveText(['Início', 'Minha conta'])
    await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
  })

  test('/login sends it to its landing page', async ({ page }) => {
    await page.goto('/login')
    await expect(page).toHaveURL('/usuarios')
  })
})

test.describe('admin', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, seedAdmin(), '/administradores')
  })

  test('sees its area, and the comum area is forbidden', async ({ page }) => {
    await expectArea(page, '/administradores', 'administradores')
    await expect(page.getByText(ADMIN_TEXT, { exact: true })).toBeVisible()
    await expectForbidden(page, '/usuarios')
  })

  test('/ goes to its area, and the header shows the admin links', async ({
    page
  }) => {
    await page.goto('/')
    await expect(page).toHaveURL('/administradores')
    await expect(headerLinks(page)).toHaveText([
      'Início',
      'Área do administrador',
      'Cadastrar usuários'
    ])
  })

  test('the forbidden screen links back to the landing page', async ({
    page
  }) => {
    await page.goto('/usuarios')
    await page
      .getByRole('link', { name: 'Ir para a minha página inicial' })
      .click()
    await expect(page).toHaveURL('/administradores')
  })
})

test.describe('old page paths', () => {
  const OLD_PATHS = ['/minha-conta', '/admin', '/cadastrar-usuarios']
  const who = {
    admin: (page: Page) => loginAs(page, seedAdmin(), '/administradores'),
    anonymous: async () => {},
    comum: (page: Page) => loginAs(page, comumUser(), '/usuarios')
  }

  for (const [name, signIn] of Object.entries(who)) {
    test(`answer 404 for ${name}`, async ({ page }) => {
      await signIn(page)
      for (const path of OLD_PATHS) {
        const response = await page.goto(path)
        expect(response?.status(), path).toBe(404)
        await expect(page).toHaveURL(path)
        await expect(page.getByTestId('area')).toHaveCount(0)
        await expect(page.getByTestId('forbidden')).toHaveCount(0)
        await expect(page.getByLabel('Login')).toHaveCount(0)
        await expect(page.getByLabel('Papel')).toHaveCount(0)
      }
    })
  }
})
