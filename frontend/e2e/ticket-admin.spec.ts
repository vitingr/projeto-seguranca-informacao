import { type Page, expect, test } from '@playwright/test'

import { setRole } from './support/db'
import { seedAdmin } from './support/env'
import { type Credentials, loginAs, submitLogin } from './support/login'
import {
  asUser,
  changeStatus,
  createTicket,
  registerComum,
  reply
} from './support/tracking'

const FORBIDDEN = 'Você não tem permissão para acessar esta página.'
const PAYLOAD = '<img src=x onerror=alert(1)> <b>negrito</b>'

const thread = (page: Page) =>
  page.getByRole('list', { name: 'Conversa do ticket' })
const items = (page: Page) => thread(page).locator(':scope > li')
const adminPath = (id: number) => `/administradores/tickets/${id}`

let author: Credentials
let colleague: Credentials
let id = 0

test.beforeAll(async ({ browser }) => {
  author = await registerComum(browser)
  colleague = await registerComum(browser)
  setRole(colleague.login, 'admin')
  await asUser(browser, author, '/usuarios', async page => {
    id = await createTicket(page, {
      dados: {
        descricao: 'Ao salvar o formulário, a página mostra erro 500.',
        passos: '1. Abra o portal\n2. Clique em Salvar',
        severidade: 'critica',
        sistema: 'Portal do aluno'
      },
      tipo: 'bug'
    })
    await reply(page, 'own', id, PAYLOAD)
  })
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await changeStatus(admin, id, 'aberto', 'em_andamento')
  })
  await asUser(browser, colleague, '/administradores', async admin => {
    await reply(admin, 'queue', id, 'Já vi esse erro antes.')
  })
})

test.describe('as admin', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, seedAdmin(), '/administradores')
  })

  test('opens a ticket from the queue, with "Usuário" in the header (AC-5.1, AC-5.2)', async ({
    page
  }) => {
    await page.goto(`/administradores?usuario=${author.login}`)
    await page
      .getByTestId('ticket-queue')
      .locator('table')
      .getByRole('link', { name: 'Bug em Portal do aluno' })
      .click()
    await expect(page).toHaveURL(
      `${adminPath(id)}?de=${encodeURIComponent(`/administradores?usuario=${author.login}`)}`
    )
    const header = page.getByTestId('ticket-header')
    await expect(header).toContainText(`Usuário:${author.login}`)
    await expect(header).toContainText('Em andamento')
    await expect(
      page
        .getByRole('tabpanel', { name: 'Detalhes' })
        .locator('dt', { hasText: /^Severidade$/ })
        .locator('xpath=following-sibling::dd[1]')
    ).toHaveText('Crítica')

    await page.getByRole('link', { name: 'Voltar para a fila' }).click()
    await expect(page).toHaveURL(`/administradores?usuario=${author.login}`)
  })

  test('shows the support view of the thread (AC-5.3)', async ({ page }) => {
    await page.goto(`${adminPath(id)}?aba=conversa`)
    await expect(items(page).nth(0)).toHaveText(
      new RegExp(`^${author.login.replaceAll('.', '\\.')} abriu o ticket`)
    )
    const fromAuthor = items(page).nth(1)
    await expect(fromAuthor).toHaveAttribute('data-side', 'other')
    await expect(fromAuthor).toContainText(author.login)
    await expect(fromAuthor.locator('[data-avatar-tint]')).toBeAttached()
    await expect(fromAuthor).toContainText(PAYLOAD)
    await expect(thread(page).locator('img, b')).toHaveCount(0)
    await expect(items(page).nth(2)).toHaveText(
      /^Você alterou o status para Em andamento/
    )
    const fromColleague = items(page).nth(3)
    await expect(fromColleague).toHaveAttribute('data-side', 'other')
    await expect(fromColleague).toContainText(colleague.login)
    await expect(fromColleague.locator('[data-support-mark]')).toBeAttached()
  })

  test('replies to the author; the author sees "Suporte" (AC-5.4)', async ({
    browser,
    page
  }) => {
    await page.goto(`${adminPath(id)}?aba=conversa`)
    const box = page.getByRole('textbox', { name: 'Mensagem' })
    await expect(box).toHaveAttribute(
      'placeholder',
      `Responder a ${author.login}…`
    )
    await expect(page.getByText(`Visível para ${author.login}`)).toBeVisible()
    await box.fill('Corrigido, pode testar de novo.')
    await page.getByRole('button', { name: 'Enviar mensagem' }).click()
    const own = items(page).last()
    await expect(own).toHaveAttribute('data-side', 'own')
    await expect(own).toContainText('Você')
    await expect(own).toContainText('Corrigido, pode testar de novo.')

    await asUser(browser, author, '/usuarios', async authorPage => {
      await authorPage.goto(`/usuarios/tickets/${id}?aba=conversa`)
      const last = authorPage
        .getByRole('list', { name: 'Conversa do ticket' })
        .locator(':scope > li')
        .last()
      await expect(last).toContainText('Suporte')
      await expect(last).toContainText('Corrigido, pode testar de novo.')
      await expect(last).not.toContainText(seedAdmin().login)
    })
  })

  test('a missing ticket shows "Ticket não encontrado." with "Voltar para a fila"', async ({
    page
  }) => {
    for (const path of [
      '/administradores/tickets/999999999',
      '/administradores/tickets/abc'
    ]) {
      await page.goto(path)
      await expect(page.getByTestId('ticket-not-found')).toHaveText(
        'Ticket não encontrado.'
      )
    }
    await page.getByRole('link', { name: 'Voltar para a fila' }).click()
    await expect(page).toHaveURL('/administradores')
  })
})

test('a comum user gets the forbidden screen (S-1)', async ({ page }) => {
  await loginAs(page, author, '/usuarios')
  await page.goto(adminPath(id))
  await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
  await expect(page.getByTestId('ticket-header')).toHaveCount(0)
})

test('without a session, the login brings the admin back to the ticket (S-15)', async ({
  page
}) => {
  await page.goto(adminPath(id))
  await expect(page).toHaveURL(
    `/login?next=${encodeURIComponent(adminPath(id))}`
  )
  await submitLogin(page, seedAdmin())
  await expect(page).toHaveURL(adminPath(id))
  await expect(page.getByTestId('ticket-header')).toContainText(author.login)
})
