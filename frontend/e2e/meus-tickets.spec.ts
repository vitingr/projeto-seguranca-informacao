import { type Page, expect, test } from '@playwright/test'

import { loginAs } from './support/login'
import {
  createTicket,
  insertOtherTickets,
  registerComum
} from './support/tracking'

// "Meus tickets" on /usuarios (spec 003 US-1, task T7.4). Each test file
// registers users of its own (001 T9.8 pattern), so the lists hold only
// what this file created.

const COMUM_TEXT = 'Página de acesso restrito aos usuários comuns'

const section = (page: Page) => page.getByTestId('my-tickets')

const titles = (page: Page) =>
  section(page).locator('table tbody th[scope="row"] a')

const statusLink = (page: Page, name: string) =>
  page
    .getByRole('navigation', { name: 'Filtrar por status' })
    .getByRole('link', { exact: true, name })

const chooseType = async (page: Page, option: string) => {
  await page.getByRole('combobox', { name: 'Tipo' }).click()
  await page.getByRole('option', { exact: true, name: option }).click()
}

const BUG = {
  dados: {
    descricao: 'Ao salvar o formulário, a página mostra erro 500.',
    passos: '1. Abra o portal\n2. Clique em Salvar',
    severidade: 'alta',
    sistema: 'Portal do aluno'
  },
  tipo: 'bug'
} as const

const OTHER = {
  dados: {
    descricao: 'Preciso da porta 8443 aberta para o servidor de testes.',
    titulo: 'Liberar porta no firewall'
  },
  tipo: 'outro'
} as const

const DEPLOY = {
  dados: {
    ambiente: 'homolog',
    projeto: 'api-pagamentos',
    repositorio: 'https://github.com/org/api-pagamentos'
  },
  tipo: 'deploy'
} as const

test.describe('a user with no ticket', () => {
  test('keeps the exact text and the profile, and sees the empty list (AC-1.1, AC-1.8)', async ({
    browser,
    page
  }) => {
    const user = await registerComum(browser)
    await loginAs(page, user, '/usuarios')
    await expect(page.getByText(COMUM_TEXT, { exact: true })).toBeVisible()
    await expect(page.getByTestId('area')).toHaveText(
      'Área liberada pelo servidor: usuarios'
    )
    await expect(page.getByTestId('profile-identity')).toContainText(user.login)
    await expect(
      section(page).getByRole('heading', { level: 2, name: 'Meus tickets' })
    ).toBeVisible()
    await expect(page.getByTestId('ticket-count')).toHaveText('0 tickets')
    await expect(section(page)).toContainText(
      'Você ainda não criou nenhum ticket.'
    )
    // The one "Novo ticket" of the page is the next action.
    await expect(page.getByRole('link', { name: 'Novo ticket' })).toHaveCount(1)
    await page.getByRole('link', { name: 'Novo ticket' }).click()
    await expect(page).toHaveURL('/usuarios/tickets/novo')
  })
})

test.describe('a user with tickets', () => {
  let author: { login: string; password: string }
  let other: { login: string; password: string }

  test.beforeAll(async ({ browser }) => {
    author = await registerComum(browser)
    other = await registerComum(browser)
    const context = await browser.newContext()
    const page = await context.newPage()
    await loginAs(page, author, '/usuarios')
    await createTicket(page, BUG)
    await createTicket(page, DEPLOY)
    await createTicket(page, OTHER)
    await context.close()

    const otherContext = await browser.newContext()
    const otherPage = await otherContext.newPage()
    await loginAs(otherPage, other, '/usuarios')
    await createTicket(otherPage, {
      dados: {
        justificativa: 'Preciso consultar os painéis de uso do mês.',
        nivel_acesso: 'leitura',
        sistema: 'Grafana'
      },
      tipo: 'acesso'
    })
    await otherContext.close()
  })

  test.beforeEach(async ({ page }) => {
    await loginAs(page, author, '/usuarios')
  })

  test('lists only their own tickets, most recently updated first (AC-1.2 to AC-1.4)', async ({
    page
  }) => {
    await expect(page.getByTestId('ticket-count')).toHaveText('3 tickets')
    await expect(titles(page)).toHaveText([
      'Liberar porta no firewall',
      'Deploy de api-pagamentos em Homologação',
      'Bug em Portal do aluno'
    ])
    await expect(section(page)).not.toContainText('Acesso a Grafana')
    const row = section(page).locator('tbody tr').first()
    await expect(row).toContainText('Outro')
    await expect(row).toContainText('Aberto')
    await expect(row.locator('time').first()).toHaveText(
      /^\d{2}\/\d{2}\/\d{4} às \d{2}:\d{2}$/
    )
    await expect(statusLink(page, 'Todos')).toHaveAttribute(
      'aria-current',
      'page'
    )
  })

  test('filters by status and type in the address, kept on reload and back (AC-1.5, AC-1.6)', async ({
    page
  }) => {
    await statusLink(page, 'Aberto').click()
    await expect(page).toHaveURL('/usuarios?status=aberto')
    await expect(statusLink(page, 'Aberto')).toHaveAttribute(
      'aria-current',
      'page'
    )
    await chooseType(page, 'Suporte em bug')
    await expect(page).toHaveURL('/usuarios?status=aberto&tipo=bug')
    await expect(titles(page)).toHaveText(['Bug em Portal do aluno'])
    await expect(page.getByTestId('ticket-count')).toHaveText('1 ticket')

    await page.reload()
    await expect(titles(page)).toHaveText(['Bug em Portal do aluno'])
    await expect(page.getByRole('combobox', { name: 'Tipo' })).toHaveText(
      'Suporte em bug'
    )

    await page.goBack()
    await expect(page).toHaveURL('/usuarios?status=aberto')
    await expect(titles(page)).toHaveCount(3)

    await chooseType(page, 'Todos')
    await expect(page).toHaveURL('/usuarios?status=aberto')
  })

  test('shows "Nenhum ticket com esses filtros." and clears them (AC-1.8)', async ({
    page
  }) => {
    await page.goto('/usuarios?status=fechado&tipo=deploy')
    await expect(section(page)).toContainText(
      'Nenhum ticket com esses filtros.'
    )
    await expect(page.getByTestId('ticket-count')).toHaveText('0 tickets')
    await section(page).getByRole('link', { name: 'Limpar filtros' }).click()
    await expect(page).toHaveURL('/usuarios')
    await expect(titles(page)).toHaveCount(3)
  })

  test('reads an unknown value in the address as "all" (AC-1.6)', async ({
    page
  }) => {
    await page.goto('/usuarios?status=xyz&tipo=nada&pagina=abc')
    await expect(statusLink(page, 'Todos')).toHaveAttribute(
      'aria-current',
      'page'
    )
    await expect(page.getByRole('combobox', { name: 'Tipo' })).toHaveText(
      'Todos'
    )
    await expect(titles(page)).toHaveCount(3)
  })
})

test('21 tickets make two pages of 20 (AC-1.7)', async ({ browser, page }) => {
  const user = await registerComum(browser)
  insertOtherTickets(user.login, 21)
  await loginAs(page, user, '/usuarios')
  await expect(page.getByTestId('ticket-count')).toHaveText('21 tickets')
  await expect(titles(page)).toHaveCount(20)
  await expect(titles(page).first()).toHaveText('Pedido 21')
  const pages = page.getByRole('navigation', { name: 'Páginas' })
  await expect(pages).toContainText('Página 1 de 2')
  await pages.getByRole('link', { name: 'Próxima' }).click()
  await expect(page).toHaveURL('/usuarios?pagina=2')
  await expect(titles(page)).toHaveText(['Pedido 1'])
  await expect(pages).toContainText('Página 2 de 2')
  await expect(pages.getByRole('link', { name: 'Próxima' })).toHaveCount(0)

  await page.goto('/usuarios?pagina=9')
  await expect(section(page)).toContainText('Esta página não existe.')
  await section(page)
    .getByRole('link', { name: 'Ir para a primeira página' })
    .click()
  await expect(page).toHaveURL('/usuarios')
})
