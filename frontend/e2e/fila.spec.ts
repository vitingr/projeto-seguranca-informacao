import { type Page, expect, test } from '@playwright/test'

import { seedAdmin } from './support/env'
import { type Credentials, loginAs } from './support/login'
import {
  asUser,
  changeStatus,
  createTicket,
  registerComum
} from './support/tracking'

const ADMIN_TEXT = 'Página de acesso restrito aos usuários administradores'
const FORBIDDEN = 'Você não tem permissão para acessar esta página.'

const queue = (page: Page) => page.getByTestId('ticket-queue')
const rows = (page: Page) => queue(page).locator('table tbody tr')

const chooseUser = async (page: Page, login: string) => {
  await page.getByRole('combobox', { name: 'Usuário' }).click()
  await page.getByRole('option', { exact: true, name: login }).click()
}

let ana: Credentials
let bruno: Credentials

test.beforeAll(async ({ browser }) => {
  ana = await registerComum(browser)
  bruno = await registerComum(browser)
  let closing = 0
  await asUser(browser, ana, '/usuarios', async page => {
    await createTicket(page, {
      dados: {
        descricao: 'Ao salvar o formulário, a página mostra erro 500.',
        passos: '1. Abra o portal\n2. Clique em Salvar',
        severidade: 'alta',
        sistema: 'Portal do aluno'
      },
      tipo: 'bug'
    })
    closing = await createTicket(page, {
      dados: {
        descricao: 'Preciso da porta 8443 aberta para o servidor de testes.',
        titulo: 'Liberar porta no firewall'
      },
      tipo: 'outro'
    })
  })
  await asUser(browser, bruno, '/usuarios', async page => {
    await createTicket(page, {
      dados: {
        justificativa: 'Preciso consultar os painéis de uso do mês.',
        nivel_acesso: 'leitura',
        sistema: 'Grafana'
      },
      tipo: 'acesso'
    })
  })
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await changeStatus(admin, closing, 'aberto', 'fechado')
  })
})

test.describe('as admin', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, seedAdmin(), '/administradores')
  })

  test('keeps the exact text and the way to /cadastro, and shows the queue', async ({
    page
  }) => {
    await expect(page.getByText(ADMIN_TEXT, { exact: true })).toBeVisible()
    await expect(page.getByTestId('area')).toHaveText(
      'Área liberada pelo servidor: administradores'
    )
    await expect(
      page
        .getByRole('navigation', { name: 'Principal' })
        .getByRole('link', { name: 'Cadastrar usuários' })
    ).toHaveAttribute('href', '/cadastro')
    await expect(
      queue(page).getByRole('heading', { level: 2, name: 'Fila de tickets' })
    ).toBeVisible()
    const headers = queue(page).locator('thead th')
    await expect(headers).toHaveText([
      '#',
      'Título',
      'Tipo',
      'Status',
      'Usuário',
      'Criado em',
      'Atualizado em'
    ])
    await expect(rows(page).first()).toContainText(ana.login)
    await expect(rows(page).first()).toContainText('Fechado')
    await expect(queue(page).getByTestId('ticket-count')).toHaveText(
      /^\d+ tickets?$/
    )
  })

  test('filters by user, type and status, kept in the address', async ({
    page
  }) => {
    const options = async () => {
      await page.getByRole('combobox', { name: 'Usuário' }).click()
      await expect(page.getByRole('option').first()).toBeVisible()
      const names = await page.getByRole('option').allTextContents()
      await page.keyboard.press('Escape')
      return names
    }
    const names = await options()
    expect(names[0]).toBe('Todos')
    expect(names.slice(1)).toEqual([...names.slice(1)].sort())
    expect(names).toContain(ana.login)

    await chooseUser(page, ana.login)
    await expect(page).toHaveURL(`/administradores?usuario=${ana.login}`)
    await expect(rows(page)).toHaveCount(2)
    await expect(queue(page)).not.toContainText(bruno.login)

    await page.getByRole('combobox', { name: 'Tipo' }).click()
    await page
      .getByRole('option', { exact: true, name: 'Suporte em bug' })
      .click()
    await expect(page).toHaveURL(
      `/administradores?tipo=bug&usuario=${ana.login}`
    )
    await expect(rows(page)).toHaveCount(1)
    await expect(rows(page).first()).toContainText('Bug em Portal do aluno')

    await page.reload()
    await expect(rows(page)).toHaveCount(1)
    await expect(page.getByRole('combobox', { name: 'Usuário' })).toHaveText(
      ana.login
    )

    await page.goto(`/administradores?status=fechado&usuario=${bruno.login}`)
    await expect(queue(page)).toContainText('Nenhum ticket com esses filtros.')
    await queue(page).getByRole('link', { name: 'Limpar filtros' }).click()
    await expect(page).toHaveURL('/administradores')
  })

  test('the summary counts every status and its links filter', async ({
    page
  }) => {
    const summary = page.getByRole('navigation', { name: 'Tickets por status' })
    await expect(summary.getByRole('link')).toHaveText([
      /^Aberto \d+$/,
      /^Em andamento \d+$/,
      /^Resolvido \d+$/,
      /^Fechado \d+$/
    ])
    await summary.getByRole('link', { name: /^Fechado/ }).click()
    await expect(page).toHaveURL('/administradores?status=fechado')
    await expect(
      summary.getByRole('link', { name: /^Fechado/ })
    ).toHaveAttribute('aria-current', 'page')
    for (const status of await rows(page).locator('[data-status]').all()) {
      await expect(status).toHaveText('Fechado')
    }
  })
})

test('a comum user gets the forbidden screen and a 403 from the API', async ({
  page
}) => {
  await loginAs(page, ana, '/usuarios')
  await page.goto('/administradores')
  await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
  await expect(queue(page)).toHaveCount(0)
  const status = await page.evaluate(
    async () => (await fetch('/api/administradores/tickets')).status
  )
  expect(status).toBe(403)
})
