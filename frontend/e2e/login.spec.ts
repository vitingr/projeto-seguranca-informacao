import { expect, test } from '@playwright/test'

import { seedAdmin } from './support/env'
import { loginAs, recordApiCalls, submitLogin } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'

test.describe('login page', () => {
  test('shows the form in Portuguese', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('html')).toHaveAttribute('lang', 'pt-BR')
    await expect(page.getByRole('heading', { name: 'Entrar' })).toBeVisible()
    await expect(page).toHaveTitle(/Entrar/)
  })

  test('empty fields show messages and send nothing', async ({ page }) => {
    const calls = recordApiCalls(page)
    await page.goto('/login')
    await page.getByRole('button', { name: 'Entrar' }).click()
    await expect(page.getByText('Informe o login.')).toBeVisible()
    await expect(page.getByText('Informe a senha.')).toBeVisible()
    expect(calls).toEqual([])
  })

  test('wrong credentials: message, login kept, password cleared', async ({
    page
  }) => {
    await page.goto('/login')
    await submitLogin(page, {
      login: 'nobody.here',
      password: 'definitely-not-the-password'
    })
    await expect(page.getByTestId('form-error')).toHaveText(
      'Login ou senha incorretos.'
    )
    await expect(page.getByLabel('Login')).toHaveValue('nobody.here')
    await expect(page.getByLabel('Senha')).toHaveValue('')
    await expect(page).toHaveURL('/login')
  })

  test('admin logs in: token fetched before and after, lands on /admin', async ({
    page
  }) => {
    const admin = seedAdmin()
    const calls = recordApiCalls(page)
    const consoleLines: string[] = []
    page.on('console', message => consoleLines.push(message.text()))

    await loginAs(page, admin, '/admin')

    // GET /api/csrf before the POST and again after it (002 D-4).
    expect(calls.slice(0, 3)).toEqual([
      'GET /api/csrf',
      'POST /api/login',
      'GET /api/csrf'
    ])
    expect(page.url()).not.toContain(admin.password)
    expect(consoleLines.join('\n')).not.toContain(admin.password)
    await expectNothingInBrowserStorage(page)
  })

  test('the button is disabled while the request is pending', async ({
    page
  }) => {
    await page.route('**/api/login', async route => {
      await new Promise(resolve => setTimeout(resolve, 800))
      await route.continue()
    })
    await page.goto('/login')
    await submitLogin(page, { login: 'nobody.here', password: 'wrong-pass' })
    await expect(page.getByRole('button', { name: 'Entrando…' })).toBeDisabled()
    await expect(page.getByTestId('form-error')).toHaveText(
      'Login ou senha incorretos.'
    )
  })

  test('next is followed only for an allowed route', async ({ page }) => {
    const admin = seedAdmin()
    await loginAs(
      page,
      admin,
      '/cadastrar-usuarios',
      '?next=%2Fcadastrar-usuarios'
    )
  })

  test('next cannot redirect elsewhere', async ({ page }) => {
    const admin = seedAdmin()
    await loginAs(page, admin, '/admin', '?next=%2F%2Fevil.example')
  })

  test('a logged-in user is sent away from /login', async ({ page }) => {
    await loginAs(page, seedAdmin(), '/admin')
    await page.goto('/login')
    await expect(page).toHaveURL('/admin')
  })

  test('the expired-session notice', async ({ page }) => {
    await page.goto('/login?next=%2Fadmin&sessao=expirada')
    await expect(page.getByTestId('form-error')).toHaveText(
      'Sua sessão expirou. Entre novamente.'
    )
  })
})
