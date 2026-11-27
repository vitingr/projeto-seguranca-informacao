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

// The "Conversa" tab of the author's ticket page (spec 003 US-3, task T8.3):
// both sides of the thread, the status lines, the position on open, the
// explicit refresh and the empty thread. Entries are created through the
// API, as the author and as the seed admin.

const OTHER = (titulo: string) =>
  ({
    dados: {
      descricao: 'Preciso da porta 8443 aberta para o servidor de testes.',
      titulo
    },
    tipo: 'outro'
  }) as const

const openConversa = async (page: Page, id: number) => {
  await page.goto(`/usuarios/tickets/${id}?aba=conversa`)
  await expect(page.getByRole('tabpanel', { name: 'Conversa' })).toBeVisible()
}

const thread = (page: Page) =>
  page.getByRole('list', { name: 'Conversa do ticket' })

/** The thread region is scrolled to its end and fits the screen. */
const expectLatestInView = async (page: Page) => {
  const region = page.getByTestId('thread-scroll')
  await expect
    .poll(() =>
      region.evaluate(
        element =>
          element.scrollHeight - element.scrollTop - element.clientHeight
      )
    )
    .toBeLessThan(2)
  await region.scrollIntoViewIfNeeded()
  await expect(thread(page).locator(':scope > li').last()).toBeInViewport()
  const box = await region.boundingBox()
  const viewport = page.viewportSize()
  expect(box && viewport && box.height <= viewport.height).toBe(true)
}

let author: Credentials
let talked: number
let long: number
let empty: number

test.beforeAll(async ({ browser }) => {
  author = await registerComum(browser)
  await asUser(browser, author, '/usuarios', async page => {
    talked = await createTicket(page, OTHER('Conversa com o suporte'))
    long = await createTicket(page, OTHER('Conversa longa'))
    empty = await createTicket(page, OTHER('Sem conversa'))
    await reply(page, 'own', talked, 'Bom dia!\nO servidor é o teste-01.')
  })
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await changeStatus(admin, talked, 'aberto', 'em_andamento')
    await reply(admin, 'queue', talked, 'Estamos olhando agora.')
  })
  await asUser(browser, author, '/usuarios', async page => {
    await reply(page, 'own', talked, 'Obrigado.')
    for (let index = 1; index <= 14; index += 1) {
      await reply(page, 'own', long, `Mensagem número ${index}`)
    }
  })
})

test.beforeEach(async ({ page }) => {
  await loginAs(page, author, '/usuarios')
})

test('shows both sides and the status lines in date order (AC-3.1, AC-3.2)', async ({
  page
}) => {
  await openConversa(page, talked)
  await expect(page.getByTestId('message-count')).toHaveText('3 mensagens')
  const items = thread(page).locator(':scope > li')
  await expect(items).toHaveCount(5)
  await expect(items.nth(0)).toHaveText(/^Você abriu o ticket/)
  await expect(items.nth(1)).toHaveAttribute('data-side', 'own')
  await expect(items.nth(1)).toContainText('Você')
  await expect(items.nth(1).locator('article p').last()).toHaveText(
    'Bom dia!\nO servidor é o teste-01.'
  )
  await expect(items.nth(2)).toHaveText(
    /^Suporte alterou o status para Em andamento/
  )
  await expect(items.nth(3)).toHaveAttribute('data-side', 'other')
  await expect(items.nth(3)).toContainText('Suporte')
  await expect(items.nth(3)).toContainText('Estamos olhando agora.')
  await expect(items.nth(3).locator('[data-support-mark]')).toBeAttached()
  await expect(items.nth(4)).toHaveAttribute('data-side', 'own')
  // The comum author never sees an admin login (S-13).
  await expect(thread(page)).not.toContainText(seedAdmin().login)
})

test('opens on the latest entry, inside a bounded scroll region (AC-3.4a)', async ({
  page
}) => {
  await openConversa(page, long)
  const items = thread(page).locator(':scope > li')
  await expect(items).toHaveCount(15)
  await expectLatestInView(page)
  // Older entries are reached by scrolling up inside the region.
  await expect(items.first()).not.toBeInViewport()
})

test('opening "Conversa" from "Detalhes" also shows the latest entry (AC-3.4a)', async ({
  page
}) => {
  await page.goto(`/usuarios/tickets/${long}`)
  await page.getByRole('tab', { name: 'Conversa' }).click()
  await expectLatestInView(page)
})

test('"Atualizar" brings what arrived meanwhile; nothing refreshes by itself (AC-3.10, BR-9)', async ({
  browser,
  page
}) => {
  await openConversa(page, talked)
  const before = await thread(page).locator(':scope > li').count()
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    await reply(admin, 'queue', talked, 'Porta liberada.')
  })
  await expect(thread(page)).not.toContainText('Porta liberada.')
  await page.getByRole('button', { name: 'Atualizar' }).click()
  await expect(thread(page)).toContainText('Porta liberada.')
  await expect(thread(page).locator(':scope > li')).toHaveCount(before + 1)
  await expect(page.getByTestId('refreshed-at')).toHaveText(
    /^Atualizado às \d{2}:\d{2}$/
  )
})

test('an empty thread shows the opening line and the invitation (AC-3.3)', async ({
  page
}) => {
  await openConversa(page, empty)
  await expect(page.getByTestId('message-count')).toHaveText('0 mensagens')
  await expect(thread(page).locator(':scope > li')).toHaveText([
    /^Você abriu o ticket/,
    'Nenhuma mensagem ainda. Escreva abaixo se quiser acrescentar algo ao ticket.'
  ])
})

test('at 360 px the panel fits the page height', async ({ page }) => {
  await page.setViewportSize({ height: 740, width: 360 })
  await openConversa(page, long)
  await expectLatestInView(page)
})
