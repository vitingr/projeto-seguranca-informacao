import { type Page, expect, test } from '@playwright/test'

import { seedAdmin } from './support/env'
import { type Credentials, loginAs, recordApiCalls } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'
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

const box = (page: Page) => page.getByRole('textbox', { name: 'Mensagem' })
const sendButton = (page: Page) =>
  page.getByRole('button', { name: 'Enviar mensagem' })
const thread = (page: Page) =>
  page.getByRole('list', { name: 'Conversa do ticket' })
const posts = (calls: string[]) =>
  calls.filter(call => /^POST \/api\/tickets\/\d+\/mensagens$/.test(call))

const openConversa = async (page: Page, id: number) => {
  await page.goto(`/usuarios/tickets/${id}?aba=conversa`)
  await expect(box(page)).toBeVisible()
}

const answer = (status: number, error: string) => ({
  body: JSON.stringify({ error, message: 'x', request_id: 'req-e2e-9' }),
  contentType: 'application/json',
  status
})

let author: Credentials
let first: number
let second: number

test.beforeAll(async ({ browser }) => {
  author = await registerComum(browser)
  await asUser(browser, author, '/usuarios', async page => {
    first = await createTicket(page, OTHER('Primeiro pedido'))
    second = await createTicket(page, OTHER('Segundo pedido'))
  })
})

test.beforeEach(async ({ page }) => {
  await loginAs(page, author, '/usuarios')
})

test('the author replies; the reply shows on the right and the ticket goes to the top (AC-3.6, BR-8)', async ({
  browser,
  page
}) => {
  const calls = recordApiCalls(page)
  await openConversa(page, first)
  await expect(page.getByText('Visível para a equipe de suporte')).toBeVisible()
  await expect(page.getByText('Ctrl+Enter para enviar')).toBeVisible()
  await box(page).fill('Bom dia!\nO servidor é o teste-01.')
  await expect(page.getByTestId('reply-counter')).toHaveText('33 / 2000')
  await sendButton(page).dblclick()
  const own = thread(page).locator('li[data-side="own"]').last()
  await expect(own).toContainText('Bom dia!\nO servidor é o teste-01.')
  await expect(box(page)).toHaveValue('')
  await expect(page.getByTestId('reply-status')).toHaveText('Mensagem enviada.')
  expect(posts(calls)).toHaveLength(1)
  await expectNothingInBrowserStorage(page)
  expect(page.url()).not.toContain('teste-01')

  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await reply(admin, 'queue', first, 'Estamos olhando.')
  })
  await page.getByRole('button', { name: 'Atualizar' }).click()
  await expect(
    thread(page).locator('li[data-side="other"]').last()
  ).toContainText('Estamos olhando.')

  await page.goto('/usuarios')
  await expect(
    page.getByTestId('my-tickets').locator('tbody tr').first()
  ).toHaveAttribute('data-ticket-id', String(first))
  await expect(
    page.getByTestId('my-tickets').locator('tbody tr').first()
  ).toContainText('Aberto')
})

test('Enter adds a line and Ctrl+Enter sends (AC-3.4)', async ({ page }) => {
  await openConversa(page, second)
  await box(page).fill('Linha um')
  await box(page).press('Enter')
  await box(page).pressSequentially('Linha dois')
  await expect(box(page)).toHaveValue('Linha um\nLinha dois')
  await box(page).press('Control+Enter')
  await expect(
    thread(page).locator('li[data-side="own"]').last()
  ).toContainText('Linha um\nLinha dois')
  await expect(box(page)).toHaveValue('')
})

test('whitespace only and 2001 characters send nothing (AC-3.5)', async ({
  page
}) => {
  const calls = recordApiCalls(page)
  await openConversa(page, second)
  await box(page).fill('   \n  ')
  await expect(sendButton(page)).toHaveAttribute('aria-disabled', 'true')
  await sendButton(page).click({ force: true })
  await box(page).press('Control+Enter')

  await box(page).fill('a'.repeat(2001))
  await expect(page.getByTestId('reply-counter')).toHaveText('2001 / 2000')
  await sendButton(page).click()
  await expect(page.getByText('Use no máximo 2000 caracteres.')).toBeVisible()
  await expect(box(page)).toHaveAttribute('aria-invalid', 'true')
  expect(posts(calls)).toEqual([])
})

for (const [name, route, message] of [
  [
    '422',
    answer(422, 'unprocessable_entity'),
    'Dados inválidos. Revise a mensagem e tente novamente.'
  ],
  [
    '429',
    answer(429, 'too_many_requests'),
    'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.'
  ],
  [
    '500',
    answer(500, 'internal'),
    'Ocorreu um erro inesperado. Tente novamente mais tarde. (código: req-e2e-9)'
  ],
  [
    'a network failure',
    null,
    'Não foi possível conectar ao servidor. Tente novamente.'
  ]
] as const) {
  test(`${name} keeps the text and shows its message (AC-3.7)`, async ({
    page
  }) => {
    await openConversa(page, second)
    await page.route('**/api/tickets/*/mensagens', async request => {
      if (request.request().method() !== 'POST') return request.continue()
      if (route === null) return request.abort()
      return request.fulfill(route)
    })
    await box(page).fill('Texto que não pode se perder.')
    await sendButton(page).click()
    await expect(page.getByTestId('send-error')).toHaveText(message)
    await expect(box(page)).toHaveValue('Texto que não pode se perder.')
  })
}

test('a ticket closed meanwhile refuses the reply and shows the closed line (AC-3.7, AC-3.8)', async ({
  browser,
  page
}) => {
  let closing = 0
  await asUser(browser, author, '/usuarios', async other => {
    closing = await createTicket(other, OTHER('Pedido que será fechado'))
  })
  await openConversa(page, closing)
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await changeStatus(admin, closing, 'aberto', 'fechado')
  })
  await box(page).fill('Ainda preciso disso.')
  await sendButton(page).click()
  await expect(page.getByTestId('send-error')).toHaveText(
    'Este ticket foi fechado e não aceita novas mensagens.'
  )
  await expect(page.getByTestId('reply-closed')).toContainText(
    'Este ticket foi fechado. Abra um novo ticket se precisar de mais ajuda.'
  )
  await expect(box(page)).toHaveCount(0)
  await expect(
    page.getByTestId('reply-closed').getByRole('link', { name: 'Novo ticket' })
  ).toHaveAttribute('href', '/usuarios/tickets/novo')
})

test('a "Fechado" ticket shows the closed line and "Novo ticket", no box (AC-3.8)', async ({
  browser,
  page
}) => {
  let closed = 0
  await asUser(browser, author, '/usuarios', async other => {
    closed = await createTicket(other, OTHER('Pedido já fechado'))
  })
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await changeStatus(admin, closed, 'aberto', 'fechado')
  })
  await page.goto(`/usuarios/tickets/${closed}?aba=conversa`)
  await expect(page.getByTestId('reply-closed')).toContainText(
    'Este ticket foi fechado. Abra um novo ticket se precisar de mais ajuda.'
  )
  await expect(box(page)).toHaveCount(0)
  await expect(sendButton(page)).toHaveCount(0)
})

test.describe('unsent text (AC-3.9)', () => {
  const dialogs = (page: Page) => {
    const seen: string[] = []
    page.on('dialog', async dialog => {
      seen.push(dialog.type())
      await dialog.dismiss()
    })
    return seen
  }

  test('a link out of the page and "Sair" show the inline bar, never a dialog', async ({
    page
  }) => {
    const seen = dialogs(page)
    await openConversa(page, second)
    await box(page).fill('Texto ainda não enviado.')
    await page.getByRole('link', { name: 'Voltar para meus tickets' }).click()
    const bar = page.getByTestId('leave-guard')
    await expect(bar).toContainText(
      'Sair sem enviar? A mensagem que você escreveu será perdida.'
    )
    await expect(
      page.getByRole('button', { name: 'Continuar no ticket' })
    ).toBeFocused()
    await expect(page).toHaveURL(`/usuarios/tickets/${second}?aba=conversa`)
    await page.getByRole('button', { name: 'Continuar no ticket' }).click()
    await expect(bar).toHaveCount(0)
    await expect(box(page)).toHaveValue('Texto ainda não enviado.')

    await page.getByRole('button', { name: 'Sair' }).click()
    await expect(bar).toBeVisible()
    await page.getByRole('button', { name: 'Continuar no ticket' }).click()

    await page.getByRole('link', { name: 'Voltar para meus tickets' }).click()
    await page.getByRole('button', { name: 'Sair sem enviar' }).click()
    await expect(page).toHaveURL('/usuarios')
    expect(seen).toEqual([])
    await expectNothingInBrowserStorage(page)
  })

  test('closing the page with typed text asks the browser first', async ({
    page
  }) => {
    const seen = dialogs(page)
    await openConversa(page, second)
    await box(page).fill('Texto ainda não enviado.')
    await page.close({ runBeforeUnload: true })
    await expect.poll(() => seen).toEqual(['beforeunload'])
  })

  test('with an empty box nothing warns', async ({ page }) => {
    const seen = dialogs(page)
    await openConversa(page, second)
    await page.getByRole('link', { name: 'Voltar para meus tickets' }).click()
    await expect(page).toHaveURL('/usuarios')
    expect(seen).toEqual([])
  })
})

test('an HTML payload is sent and shown as text (S-10)', async ({ page }) => {
  const payload = '<img src=x onerror=alert(1)> <b>negrito</b>'
  await openConversa(page, second)
  await box(page).fill(payload)
  await sendButton(page).click()
  await expect(
    thread(page).locator('li[data-side="own"]').last()
  ).toContainText(payload)
  await expect(thread(page).locator('img, b')).toHaveCount(0)
})
