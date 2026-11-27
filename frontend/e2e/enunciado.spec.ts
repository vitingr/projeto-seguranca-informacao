import { randomBytes } from 'node:crypto'

import { type BrowserContext, type Page, expect, test } from '@playwright/test'

import { generateComumCredentials, seedAdmin } from './support/env'
import { type Credentials, loginAs, submitLogin } from './support/login'
import { registerComum } from './support/tracking'

// A guided demo of the assignment requirements, one describe per item, for
// the class presentation (`pnpm demo` / `make demo`: headed, slowed down).
// The seed admin only bootstraps: every comum user is created here.

const ADMIN_TEXT = 'Página de acesso restrito aos usuários administradores'
const COMUM_TEXT = 'Página de acesso restrito aos usuários comuns'
const FORBIDDEN = 'Você não tem permissão para acessar esta página.'
const LOGIN_FAILED = 'Login ou senha incorretos.'
const ERROR_KEYS = ['error', 'message', 'request_id']

interface Answer {
  body: Record<string, unknown>
  status: number
}

const sessionCookie = async (context: BrowserContext) =>
  (await context.cookies()).find(c => c.name === '__Host-session')

/** A request from the page, with the CSRF token unless `csrf` is false. */
const apiCall = (
  page: Page,
  method: string,
  url: string,
  { body, csrf = true }: { body?: unknown; csrf?: boolean } = {}
) =>
  page.evaluate<Answer, [string, string, unknown, boolean]>(
    async ([method, url, body, csrf]) => {
      const headers: Record<string, string> = {}
      if (body !== undefined) headers['Content-Type'] = 'application/json'
      if (csrf) {
        headers['X-CSRF-Token'] = (
          await (await fetch('/api/csrf')).json()
        ).csrf_token
      }
      const res = await fetch(url, {
        body: body === undefined ? undefined : JSON.stringify(body),
        headers,
        method
      })
      return { body: await res.json(), status: res.status }
    },
    [method, url, body, csrf]
  )

/** Only the generic error shape reaches the client. */
const expectGenericError = (answer: Answer, status: number) => {
  expect(answer.status).toBe(status)
  expect(Object.keys(answer.body).sort()).toEqual(ERROR_KEYS)
}

const fillCadastro = async (
  page: Page,
  { login, papel, senha }: { login: string; papel: string; senha: string }
) => {
  await page.getByLabel('Login').fill(login)
  await page.getByLabel('Senha').fill(senha)
  await page.getByLabel('Papel').selectOption(papel)
  await page.getByRole('button', { name: 'Cadastrar' }).click()
}

const expectForbidden = async (page: Page, path: string) => {
  await page.goto(path)
  await expect(page).toHaveURL(path)
  await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
}

test.describe('1. A página padrão é a de login, com os campos de login e senha e o botão de entrar', () => {
  test('"/" redireciona para /login, que mostra login, senha e o botão', async ({
    page
  }) => {
    await page.goto('/')
    await expect(page).toHaveURL('/login')
    await expect(page.getByLabel('Login')).toBeVisible()
    await expect(page.getByLabel('Senha')).toBeVisible()
    await expect(page.getByLabel('Senha')).toHaveAttribute('type', 'password')
    await expect(page.getByRole('button', { name: 'Entrar' })).toBeVisible()
  })
})

test.describe('2. O administrador faz login e vai para /administradores', () => {
  test('mostra o texto exato da página e o botão de sair', async ({ page }) => {
    await loginAs(page, seedAdmin(), '/administradores')
    await expect(page.getByText(ADMIN_TEXT, { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
  })
})

test.describe('3. Em /cadastro o administrador cadastra usuários com login, senha e tipo de usuário', () => {
  test('a página tem login, senha, tipo de usuário e o botão; o admin cadastra um usuário comum', async ({
    page
  }) => {
    const user = generateComumCredentials(randomBytes)
    await loginAs(page, seedAdmin(), '/administradores')
    await page.goto('/cadastro')
    await expect(page.getByLabel('Login')).toBeVisible()
    await expect(page.getByLabel('Senha')).toBeVisible()
    await expect(page.getByLabel('Papel')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Cadastrar' })).toBeVisible()

    await fillCadastro(page, {
      login: user.login,
      papel: 'comum',
      senha: user.password
    })
    await expect(page.getByTestId('form-success')).toHaveText(
      `Usuário ${user.login} cadastrado como comum.`
    )
  })
})

test.describe('4. O usuário comum faz login e vai para /usuarios', () => {
  let comum: Credentials

  test.beforeAll(async ({ browser }) => {
    comum = await registerComum(browser)
  })

  test('mostra o texto exato da página e o botão de sair', async ({ page }) => {
    await loginAs(page, comum, '/usuarios')
    await expect(page.getByText(COMUM_TEXT, { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sair' })).toBeVisible()
  })
})

test.describe('5. Sem login, as páginas restritas redirecionam para /login', () => {
  for (const path of ['/cadastro', '/administradores', '/usuarios']) {
    test(`${path} vai para /login`, async ({ page }) => {
      await page.goto(path)
      await expect(page).toHaveURL(`/login?next=${encodeURIComponent(path)}`)
      await expect(page.getByRole('button', { name: 'Entrar' })).toBeVisible()
    })
  }
})

test.describe('6. Controle de acesso por papel: cada papel só acessa as próprias páginas e APIs', () => {
  let comum: Credentials

  test.beforeAll(async ({ browser }) => {
    comum = await registerComum(browser)
  })

  test('o usuário comum tem acesso negado em /administradores e /cadastro', async ({
    page
  }) => {
    await loginAs(page, comum, '/usuarios')
    await expectForbidden(page, '/administradores')
    await expectForbidden(page, '/cadastro')
    await expect(page.getByLabel('Papel')).toHaveCount(0)
  })

  test('o administrador tem acesso negado em /usuarios', async ({ page }) => {
    await loginAs(page, seedAdmin(), '/administradores')
    await expectForbidden(page, '/usuarios')
  })

  test('chamadas diretas às APIs do outro papel devolvem 403', async ({
    browser
  }) => {
    const intruder = generateComumCredentials(randomBytes)
    const comumContext = await browser.newContext()
    const adminContext = await browser.newContext()
    try {
      const asComum = await comumContext.newPage()
      await loginAs(asComum, comum, '/usuarios')
      expectGenericError(
        await apiCall(asComum, 'GET', '/api/administradores'),
        403
      )
      expectGenericError(
        await apiCall(asComum, 'POST', '/api/cadastro', {
          body: {
            login: intruder.login,
            papel: 'admin',
            senha: intruder.password
          }
        }),
        403
      )

      const asAdmin = await adminContext.newPage()
      await loginAs(asAdmin, seedAdmin(), '/administradores')
      expectGenericError(await apiCall(asAdmin, 'GET', '/api/usuarios'), 403)
    } finally {
      await comumContext.close()
      await adminContext.close()
    }
  })
})

test.describe('7. Proteção contra injeção (SQL e XSS)', () => {
  test(`"' OR '1'='1" no login não entra e mostra só a mensagem genérica`, async ({
    page
  }) => {
    for (const login of ["' OR '1'='1", `${seedAdmin().login}' --`]) {
      await page.goto('/login')
      await submitLogin(page, { login, password: "' OR '1'='1" })
      await expect(page.getByTestId('form-error')).toHaveText(LOGIN_FAILED)
      await expect(page).toHaveURL('/login')
      await expect(page.locator('body')).not.toContainText(
        /sql|syntax|postgres|pq:/i
      )
    }
    expectGenericError(await apiCall(page, 'GET', '/api/me'), 401)
  })

  test('"<script>" no cadastro não executa', async ({ page }) => {
    const script = '<script>window.__xss = true; alert(1)</script>'
    const dialogs: string[] = []
    page.on('dialog', async dialog => {
      dialogs.push(dialog.message())
      await dialog.dismiss()
    })
    await loginAs(page, seedAdmin(), '/administradores')
    await page.goto('/cadastro')

    // The form refuses it, and so does the server without the form.
    await fillCadastro(page, {
      login: script,
      papel: 'comum',
      senha: 'x'.repeat(20)
    })
    await expect(
      page.getByText(
        'Use de 3 a 64 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado.'
      )
    ).toBeVisible()
    await expect(page.getByTestId('form-success')).toHaveCount(0)
    const direct = await apiCall(page, 'POST', '/api/cadastro', {
      body: { login: script, papel: 'comum', senha: 'x'.repeat(20) }
    })
    expect(direct.status).toBeGreaterThanOrEqual(400)
    expect(direct.status).toBeLessThan(500)
    expect(Object.keys(direct.body).sort()).toEqual(ERROR_KEYS)

    expect(
      await page.evaluate(() => (window as { __xss?: boolean }).__xss)
    ).toBeUndefined()
    expect(dialogs).toEqual([])
  })
})

test.describe('8. Proteção contra CSRF', () => {
  test('POST sem o token CSRF devolve 403', async ({ page }) => {
    await page.goto('/login')
    const admin = seedAdmin()
    expectGenericError(
      await apiCall(page, 'POST', '/api/login', {
        body: { login: admin.login, senha: admin.password },
        csrf: false
      }),
      403
    )

    await loginAs(page, admin, '/administradores')
    const user = generateComumCredentials(randomBytes)
    expectGenericError(
      await apiCall(page, 'POST', '/api/cadastro', {
        body: { login: user.login, papel: 'comum', senha: user.password },
        csrf: false
      }),
      403
    )
  })
})

test.describe('9. Gerenciamento de sessão', () => {
  test('o cookie __Host-session é HttpOnly, Secure e SameSite', async ({
    context,
    page
  }) => {
    await loginAs(page, seedAdmin(), '/administradores')
    const cookie = await sessionCookie(context)
    expect(cookie).toMatchObject({ httpOnly: true, path: '/', secure: true })
    expect(['Lax', 'Strict']).toContain(cookie?.sameSite)
    expect(await page.evaluate(() => document.cookie)).toBe('')
  })

  test('o valor do cookie muda depois do login', async ({ context, page }) => {
    await page.goto('/login')
    // Asking for a CSRF token opens the anonymous session.
    await page.evaluate(() => fetch('/api/csrf'))
    const before = await sessionCookie(context)
    expect(before).toBeDefined()

    await submitLogin(page, seedAdmin())
    await expect(page).toHaveURL('/administradores')
    const after = await sessionCookie(context)
    expect(after?.value).toBeDefined()
    expect(after?.value).not.toBe(before?.value)
  })

  test('depois do logout, as páginas restritas voltam para /login', async ({
    context,
    page
  }) => {
    await loginAs(page, seedAdmin(), '/administradores')
    await page.getByRole('button', { name: 'Sair' }).click()
    await expect(page).toHaveURL('/login')
    expect(await sessionCookie(context)).toBeUndefined()

    for (const path of ['/administradores', '/cadastro']) {
      await page.goto(path)
      await expect(page).toHaveURL(`/login?next=${encodeURIComponent(path)}`)
    }
  })
})

test.describe('10. Tratamento de erros sem expor detalhes internos', () => {
  test('rota inexistente da API devolve só {error, message, request_id}', async ({
    page
  }) => {
    await page.goto('/login')
    const answer = await apiCall(page, 'GET', '/api/nao-existe', {
      csrf: false
    })
    expectGenericError(answer, 404)
    expect(answer.body.error).toBe('not_found')
  })
})
