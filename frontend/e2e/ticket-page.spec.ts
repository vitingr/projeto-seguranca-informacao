import { type Page, expect, test } from '@playwright/test'

import { seedAdmin } from './support/env'
import { type Credentials, loginAs, submitLogin } from './support/login'
import {
  type TicketBody,
  asUser,
  createTicket,
  registerComum
} from './support/tracking'

// The ticket page of a comum user (spec 003 US-2, task T8.1): header, tabs
// and "Detalhes", the not-found screen and the way back to the list.

const NOT_FOUND = 'Ticket não encontrado.'
const FORBIDDEN = 'Você não tem permissão para acessar esta página.'
const PAYLOAD = '<img src=x onerror=alert(1)> <script>alert(1)</script>'

const TICKETS: { body: TicketBody; rows: [string, string][]; title: string }[] =
  [
    {
      body: {
        dados: {
          justificativa: 'Preciso consultar os painéis\nde uso do mês.',
          nivel_acesso: 'escrita',
          sistema: 'Grafana'
        },
        tipo: 'acesso'
      },
      rows: [
        ['Sistema ou recurso', 'Grafana'],
        ['Nível de acesso', 'Escrita'],
        ['Justificativa', 'Preciso consultar os painéis\nde uso do mês.']
      ],
      title: 'Acesso a Grafana'
    },
    {
      body: {
        dados: {
          descricao: 'Ao salvar o formulário, a página mostra erro 500.',
          passos: '1. Abra o portal\n2. Clique em Salvar',
          severidade: 'critica',
          sistema: 'Portal do aluno'
        },
        tipo: 'bug'
      },
      rows: [
        ['Sistema afetado', 'Portal do aluno'],
        ['Severidade', 'Crítica'],
        ['Descrição', 'Ao salvar o formulário, a página mostra erro 500.'],
        ['Passos para reproduzir', '1. Abra o portal\n2. Clique em Salvar']
      ],
      title: 'Bug em Portal do aluno'
    },
    {
      body: {
        dados: {
          ambiente: 'prod',
          projeto: 'api-pagamentos',
          repositorio: 'https://github.com/org/api-pagamentos'
        },
        tipo: 'deploy'
      },
      rows: [
        ['Nome do projeto', 'api-pagamentos'],
        ['Repositório', 'https://github.com/org/api-pagamentos'],
        ['Ambiente', 'Produção'],
        ['Observações', 'Não informado']
      ],
      title: 'Deploy de api-pagamentos em Produção'
    },
    {
      body: {
        dados: { descricao: PAYLOAD, titulo: 'Liberar porta no firewall' },
        tipo: 'outro'
      },
      rows: [['Descrição', PAYLOAD]],
      title: 'Liberar porta no firewall'
    }
  ]

const rowValue = (page: Page, label: string) =>
  page
    .getByRole('tabpanel', { name: 'Detalhes' })
    .locator('dt', { hasText: new RegExp(`^${label}$`) })
    .locator('xpath=following-sibling::dd[1]')

const ticketPath = (id: number) => `/usuarios/tickets/${id}`

let author: Credentials
let stranger: Credentials
const ids: number[] = []

test.beforeAll(async ({ browser }) => {
  author = await registerComum(browser)
  stranger = await registerComum(browser)
  await asUser(browser, author, '/usuarios', async page => {
    for (const { body } of TICKETS) ids.push(await createTicket(page, body))
  })
})

test.describe('as the author', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, author, '/usuarios')
  })

  for (const [index, { body, rows, title }] of TICKETS.entries()) {
    test(`a "${body.tipo}" ticket shows its header and every field (AC-2.1, AC-2.2)`, async ({
      page
    }) => {
      await page.goto(ticketPath(ids[index]))
      const header = page.getByTestId('ticket-header')
      await expect(header).toContainText(`#${ids[index]}`)
      await expect(
        page.getByRole('heading', { level: 1, name: title })
      ).toBeFocused()
      await expect(header).toContainText('Aberto')
      await expect(header).toContainText('Criado em:')
      await expect(header).toContainText('Atualizado em:')
      await expect(page.getByRole('tab', { name: 'Detalhes' })).toHaveAttribute(
        'aria-selected',
        'true'
      )
      await expect(rowValue(page, 'Número')).toHaveText(`#${ids[index]}`)
      await expect(rowValue(page, 'Título')).toHaveText(title)
      for (const [label, value] of rows) {
        await expect(rowValue(page, label)).toHaveText(value)
      }
    })
  }

  test('shows an HTML payload as text (S-10)', async ({ page }) => {
    await page.goto(ticketPath(ids[3]))
    await expect(rowValue(page, 'Descrição')).toHaveText(PAYLOAD)
    await expect(page.locator('main img, main script')).toHaveCount(0)
  })

  test('keeps the tab in the address, through reload and back (AC-2.3)', async ({
    page
  }) => {
    await page.goto(ticketPath(ids[0]))
    const conversa = page.getByRole('tab', { name: 'Conversa' })
    await conversa.click()
    await expect(page).toHaveURL(`${ticketPath(ids[0])}?aba=conversa`)
    await expect(page.getByRole('tabpanel', { name: 'Conversa' })).toBeVisible()

    await page.reload()
    await expect(conversa).toHaveAttribute('aria-selected', 'true')

    // Arrow keys move between the tabs; Enter selects (WAI-ARIA tabs).
    await conversa.focus()
    await page.keyboard.press('ArrowLeft')
    await expect(page.getByRole('tab', { name: 'Detalhes' })).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(ticketPath(ids[0]))

    // Back from the list returns to the tab that was open.
    await conversa.click()
    await expect(page).toHaveURL(`${ticketPath(ids[0])}?aba=conversa`)
    await page.getByRole('link', { name: 'Voltar para meus tickets' }).click()
    await expect(page).toHaveURL('/usuarios')
    await page.goBack()
    await expect(page).toHaveURL(`${ticketPath(ids[0])}?aba=conversa`)
    await expect(conversa).toHaveAttribute('aria-selected', 'true')
  })

  test('"Voltar para meus tickets" keeps the filters of the list (AC-2.5)', async ({
    page
  }) => {
    await page.goto('/usuarios?tipo=bug')
    await page
      .getByTestId('my-tickets')
      .locator('table')
      .getByRole('link', { name: 'Bug em Portal do aluno' })
      .click()
    await expect(page).toHaveURL(
      `${ticketPath(ids[1])}?de=${encodeURIComponent('/usuarios?tipo=bug')}`
    )
    await page.getByRole('link', { name: 'Voltar para meus tickets' }).click()
    await expect(page).toHaveURL('/usuarios?tipo=bug')
  })

  for (const [name, path] of [
    ['a missing number', '/usuarios/tickets/999999999'],
    ['a malformed number', '/usuarios/tickets/abc']
  ]) {
    test(`${name} shows "Ticket não encontrado." (AC-2.4)`, async ({
      page
    }) => {
      await page.goto(path)
      await expect(page.getByTestId('ticket-not-found')).toHaveText(NOT_FOUND)
      await page.getByRole('link', { name: 'Voltar para meus tickets' }).click()
      await expect(page).toHaveURL('/usuarios')
    })
  }
})

test('another user gets the same "Ticket não encontrado." (AC-2.4, S-2)', async ({
  page
}) => {
  await loginAs(page, stranger, '/usuarios')
  await page.goto(ticketPath(ids[0]))
  await expect(page.getByTestId('ticket-not-found')).toHaveText(NOT_FOUND)
  await expect(page.locator('main')).not.toContainText('Grafana')
})

test('an admin gets the forbidden screen (S-1)', async ({ page }) => {
  await loginAs(page, seedAdmin(), '/administradores')
  await page.goto(ticketPath(ids[0]))
  await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
  await expect(page.locator('main')).toHaveCount(1)
  await expect(page.locator('main')).not.toContainText('Grafana')
})

test('without a session, the login brings the user back to the ticket (S-15)', async ({
  page
}) => {
  await page.goto(ticketPath(ids[0]))
  await expect(page).toHaveURL(
    `/login?next=${encodeURIComponent(ticketPath(ids[0]))}`
  )
  await submitLogin(page, author)
  await expect(page).toHaveURL(ticketPath(ids[0]))
  await expect(
    page.getByRole('heading', { level: 1, name: 'Acesso a Grafana' })
  ).toBeVisible()
})
