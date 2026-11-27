import { type Page, expect, test } from '@playwright/test'

import { seedAdmin } from './support/env'
import { type Credentials, loginAs } from './support/login'
import {
  asUser,
  changeStatus,
  createTicket,
  registerComum,
  reply
} from './support/tracking'

const OTHER = (titulo: string) =>
  ({
    dados: {
      descricao: 'Preciso da porta 8443 aberta para o servidor de testes.',
      titulo
    },
    tipo: 'outro'
  }) as const

const adminPath = (id: number) => `/administradores/tickets/${id}`
const trigger = (page: Page) => page.getByRole('button', { name: /^Status:/ })
const header = (page: Page) => page.getByTestId('ticket-header')
const statusLines = (page: Page) =>
  page
    .getByRole('list', { name: 'Conversa do ticket' })
    .locator('li[data-event="status_change"]')

const menuRows = async (page: Page) => {
  await trigger(page).click()
  await expect(page.getByRole('menu')).toBeVisible()
  const rows = await page.getByRole('menuitem').allTextContents()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('menu')).toHaveCount(0)
  return rows
}

const choose = async (page: Page, label: string) => {
  await trigger(page).click()
  await page.getByRole('menuitem', { exact: true, name: label }).click()
}

let author: Credentials
let path = 0
let stale = 0
let order = 0

test.beforeAll(async ({ browser }) => {
  author = await registerComum(browser)
  await asUser(browser, author, '/usuarios', async page => {
    path = await createTicket(page, OTHER('Caminho completo de status'))
    stale = await createTicket(page, OTHER('Status alterado por outra pessoa'))
    order = await createTicket(page, OTHER('Ordem das listas'))
  })
})

test.beforeEach(async ({ page }) => {
  await loginAs(page, seedAdmin(), '/administradores')
})

test('moves through every allowed status, with a status line each (AC-6.1, AC-6.2, AC-6.5)', async ({
  browser,
  page
}) => {
  const dialogs: string[] = []
  page.on('dialog', async dialog => {
    dialogs.push(dialog.type())
    await dialog.dismiss()
  })
  await page.goto(`${adminPath(path)}?aba=conversa`)

  expect(await menuRows(page)).toEqual(['Aberto', 'Em andamento', 'Fechado'])

  const steps: [string, string[]][] = [
    ['Em andamento', ['Em andamento', 'Resolvido', 'Fechado']],
    ['Resolvido', ['Resolvido', 'Em andamento', 'Fechado']],
    ['Em andamento', ['Em andamento', 'Resolvido', 'Fechado']]
  ]
  for (const [index, [label, next]] of steps.entries()) {
    await choose(page, label)
    await expect(page.getByTestId('status-status')).toHaveText(
      `Status alterado para ${label}.`
    )
    await expect(trigger(page)).toContainText(`Status: ${label}`)
    await expect(statusLines(page)).toHaveCount(index + 1)
    await expect(statusLines(page).last()).toContainText(
      `Você alterou o status para ${label}`
    )
    expect(await menuRows(page)).toEqual(next)
  }

  await choose(page, 'Fechado')
  const confirmation = page.getByTestId('close-confirmation')
  await expect(confirmation).toContainText(
    'Fechar o ticket? Ninguém poderá enviar novas mensagens.'
  )
  await confirmation.getByRole('button', { name: 'Cancelar' }).click()
  await expect(confirmation).toHaveCount(0)
  await expect(trigger(page)).toContainText('Status: Em andamento')

  await choose(page, 'Fechado')
  await page.getByRole('button', { name: 'Fechar ticket' }).click()
  await expect(header(page)).toContainText('Fechado')
  await expect(trigger(page)).toHaveCount(0)
  await expect(statusLines(page)).toHaveCount(4)
  await expect(page.getByTestId('reply-closed')).toBeVisible()
  await expect(page.getByRole('textbox', { name: 'Mensagem' })).toHaveCount(0)
  expect(dialogs).toEqual([])

  await asUser(browser, author, '/usuarios', async authorPage => {
    await authorPage.goto(`/usuarios/tickets/${path}?aba=conversa`)
    await expect(authorPage.getByTestId('ticket-header')).toContainText(
      'Fechado'
    )
    await expect(authorPage.getByTestId('reply-closed')).toBeVisible()
    await expect(authorPage.getByTestId('status-action')).toHaveCount(0)
    await expect(
      authorPage
        .getByRole('list', { name: 'Conversa do ticket' })
        .locator('li[data-event="status_change"]')
        .last()
    ).toContainText('Suporte alterou o status para Fechado')
  })
})

test('a change made meanwhile shows the stale message and the fresh status (AC-6.3)', async ({
  browser,
  page
}) => {
  await page.goto(adminPath(stale))
  await expect(trigger(page)).toContainText('Status: Aberto')
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await changeStatus(admin, stale, 'aberto', 'em_andamento')
  })
  await choose(page, 'Em andamento')
  await expect(page.getByTestId('status-error')).toHaveText(
    'O status deste ticket foi alterado por outra pessoa. A página foi atualizada; confira antes de tentar de novo.'
  )
  await expect(trigger(page)).toContainText('Status: Em andamento')
  await expect(header(page)).toContainText('Em andamento')
})

test('a status change and a reply move the ticket to the top of both lists; a reply keeps the status (BR-8)', async ({
  browser,
  page
}) => {
  const firstRow = (target: Page, list: string) =>
    target.getByTestId(list).locator('tbody tr').first()

  await asUser(browser, author, '/usuarios', async authorPage => {
    await reply(authorPage, 'own', stale, 'Oi')
  })
  await page.goto(adminPath(order))
  await choose(page, 'Em andamento')
  await expect(trigger(page)).toContainText('Status: Em andamento')
  await page.goto('/administradores')
  await expect(firstRow(page, 'ticket-queue')).toHaveAttribute(
    'data-ticket-id',
    String(order)
  )

  await asUser(browser, author, '/usuarios', async authorPage => {
    await expect(firstRow(authorPage, 'my-tickets')).toHaveAttribute(
      'data-ticket-id',
      String(order)
    )
    await reply(authorPage, 'own', stale, 'Mais uma informação.')
    await authorPage.reload()
    await expect(firstRow(authorPage, 'my-tickets')).toHaveAttribute(
      'data-ticket-id',
      String(stale)
    )
    await expect(firstRow(authorPage, 'my-tickets')).toContainText(
      'Em andamento'
    )
  })
  await page.reload()
  await expect(firstRow(page, 'ticket-queue')).toHaveAttribute(
    'data-ticket-id',
    String(stale)
  )
})
