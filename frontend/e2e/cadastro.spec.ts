import { randomBytes } from 'node:crypto'

import { type Page, expect, test } from '@playwright/test'

import { setRole } from './support/db'
import { comumUser, generateComumCredentials, seedAdmin } from './support/env'
import { loginAs, recordApiCalls } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'

const fillCadastro = async (
  page: Page,
  { login, papel, senha }: { login: string; papel?: string; senha: string }
) => {
  await page.getByLabel('Login').fill(login)
  await page.getByLabel('Senha').fill(senha)
  if (papel) await page.getByLabel('Papel').selectOption(papel)
  await page.getByRole('button', { name: 'Cadastrar' }).click()
}

const newUser = () => generateComumCredentials(randomBytes)

test.describe('registration as admin', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, seedAdmin(), '/admin')
    await page.goto('/cadastrar-usuarios')
  })

  test('registers a user, then the same login is taken', async ({ page }) => {
    const user = newUser()
    // Typed in upper case: the backend answers with the normalized login.
    await fillCadastro(page, {
      login: user.login.toUpperCase(),
      papel: 'comum',
      senha: user.password
    })
    await expect(page.getByTestId('form-success')).toHaveText(
      `Usuário ${user.login} cadastrado como comum.`
    )
    await expect(page.getByLabel('Login')).toHaveValue('')
    await expect(page.getByLabel('Senha')).toHaveValue('')

    await fillCadastro(page, {
      login: user.login,
      papel: 'admin',
      senha: user.password
    })
    await expect(page.getByText('Este login já está em uso.')).toBeVisible()
    await expect(page.getByTestId('form-success')).toHaveCount(0)
    await expectNothingInBrowserStorage(page)
  })

  test('client rules show field messages and send nothing', async ({
    page
  }) => {
    const calls = recordApiCalls(page)
    await fillCadastro(page, { login: 'ab', senha: 'short' })
    await expect(
      page.getByText(
        'Use de 3 a 64 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado.'
      )
    ).toBeVisible()
    await expect(
      page.getByText('A senha deve ter de 15 a 128 caracteres.')
    ).toBeVisible()
    await expect(page.getByText('Escolha um papel.')).toBeVisible()
    expect(calls.filter(c => c === 'POST /api/cadastro')).toEqual([])
  })

  test('an expired session ends on the login page, nothing registered', async ({
    context,
    page
  }) => {
    // A first submission (409: the login is taken) leaves the CSRF token in
    // memory, as during normal use. Nothing is created.
    await fillCadastro(page, {
      login: seedAdmin().login,
      papel: 'comum',
      senha: 'x'.repeat(15)
    })
    await expect(page.getByText('Este login já está em uso.')).toBeVisible()

    const [session] = (await context.cookies()).filter(
      c => c.name === '__Host-session'
    )
    expect(session).toBeDefined()

    // The session is deleted on the server (as an idle timeout would do),
    // but this tab still holds the old cookie.
    const other = await context.newPage()
    await other.goto('/')
    await other.evaluate(async () => {
      const token = (await (await fetch('/api/csrf')).json()).csrf_token
      await fetch('/api/logout', {
        headers: { 'X-CSRF-Token': token },
        method: 'POST'
      })
    })
    await other.close()
    await context.addCookies([session!])

    const user = newUser()
    const calls = recordApiCalls(page)
    await fillCadastro(page, {
      login: user.login,
      papel: 'comum',
      senha: user.password
    })

    // 403 (no session) → new token → retry → 401 → login with the notice.
    await expect(page).toHaveURL(
      '/login?next=%2Fcadastrar-usuarios&sessao=expirada'
    )
    await expect(page.getByTestId('form-error')).toHaveText(
      'Sua sessão expirou. Entre novamente.'
    )
    expect(calls.filter(c => c === 'POST /api/cadastro')).toHaveLength(2)

    // The user was not created: the same login can still be registered.
    await loginAs(
      page,
      seedAdmin(),
      '/cadastrar-usuarios',
      '?next=%2Fcadastrar-usuarios'
    )
    await fillCadastro(page, {
      login: user.login,
      papel: 'comum',
      senha: user.password
    })
    await expect(page.getByTestId('form-success')).toBeVisible()
  })
})

test('registration is forbidden for comum and needs a session', async ({
  browser,
  page
}) => {
  const anonymous = await browser.newPage()
  await anonymous.goto('/cadastrar-usuarios')
  await expect(anonymous).toHaveURL('/login?next=%2Fcadastrar-usuarios')
  await anonymous.close()

  await loginAs(page, comumUser(), '/minha-conta')
  await page.goto('/cadastrar-usuarios')
  await expect(page.getByTestId('forbidden')).toBeVisible()
  await expect(page.getByLabel('Papel')).toHaveCount(0)
})

test('a role change applies on the next page load, with no new login', async ({
  page
}) => {
  const comum = comumUser()
  await loginAs(page, comum, '/minha-conta')
  await page.goto('/admin')
  await expect(page.getByTestId('forbidden')).toBeVisible()
  try {
    setRole(comum.login, 'admin')
    await page.goto('/admin')
    await expect(page.getByTestId('area')).toContainText('administradores')
  } finally {
    setRole(comum.login, 'comum')
  }
  await page.goto('/admin')
  await expect(page.getByTestId('forbidden')).toBeVisible()
})
