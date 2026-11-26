import { type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs, submitLogin } from './support/login'

// The ticket builder (spec 001-ticket-builder) at /usuarios/tickets/novo.

const BUILDER = '/usuarios/tickets/novo'
const FORBIDDEN = 'Você não tem permissão para acessar esta página.'

const typeGroup = (page: Page) =>
  page.getByRole('radiogroup', { name: 'Qual é o tipo do pedido?' })

const stepHeading = (page: Page, name: string) =>
  page.getByRole('heading', { level: 2, name })

test.describe('opening the builder', () => {
  test('comum opens it from "Novo ticket" on /usuarios (AC-1.1)', async ({
    page
  }) => {
    await loginAs(page, comumUser(), '/usuarios')
    await page.getByRole('link', { name: 'Novo ticket' }).click()
    await expect(page).toHaveURL(BUILDER)
    await expect(
      page.getByRole('heading', { level: 1, name: 'Novo ticket' })
    ).toBeVisible()
    await expect(stepHeading(page, 'Qual é o tipo do pedido?')).toBeVisible()
  })

  test('admin sees no link to it and gets the forbidden screen (AC-1.2)', async ({
    page
  }) => {
    await loginAs(page, seedAdmin(), '/administradores')
    for (const path of ['/administradores', '/cadastro', '/usuarios']) {
      await page.goto(path)
      await expect(page.getByRole('link', { name: 'Novo ticket' })).toHaveCount(
        0
      )
      await expect(page.locator(`a[href="${BUILDER}"]`)).toHaveCount(0)
    }
    await page.goto(BUILDER)
    await expect(page).toHaveURL(BUILDER)
    await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
    await expect(typeGroup(page)).toHaveCount(0)
  })

  test('anonymous goes to login and comes back after it (AC-1.3)', async ({
    page
  }) => {
    await page.goto(BUILDER)
    await expect(page).toHaveURL(`/login?next=${encodeURIComponent(BUILDER)}`)
    await submitLogin(page, comumUser())
    await expect(page).toHaveURL(BUILDER)
    await expect(typeGroup(page)).toBeVisible()
  })
})

test.describe('type step', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
    await page.goto(BUILDER)
  })

  test('shows the four types with none selected (AC-2.1)', async ({ page }) => {
    const radios = typeGroup(page).getByRole('radio')
    await expect(radios).toHaveCount(4)
    for (const name of [
      'Solicitar acesso',
      'Suporte em bug',
      'Deploy de projeto',
      'Outro'
    ]) {
      await expect(
        typeGroup(page).getByRole('radio', { name })
      ).toHaveAttribute('aria-checked', 'false')
    }
    await expect(
      page.getByRole('navigation', { name: 'Etapas do ticket' })
    ).toContainText('TipoEscolha do pedido')
  })

  test('continuing with no type stays and explains (AC-2.2)', async ({
    page
  }) => {
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(page.getByText('Escolha um tipo de ticket.')).toBeVisible()
    await expect(stepHeading(page, 'Qual é o tipo do pedido?')).toBeVisible()
    // The focus goes to the group (AC-3.5).
    await expect(typeGroup(page).getByRole('radio').first()).toBeFocused()
  })

  test('works with the keyboard alone (AC-2.5)', async ({ page }) => {
    const radio = (name: string) => typeGroup(page).getByRole('radio', { name })
    await radio('Solicitar acesso').focus()
    await page.keyboard.press('Space')
    await expect(radio('Solicitar acesso')).toHaveAttribute(
      'aria-checked',
      'true'
    )
    await page.keyboard.press('ArrowDown')
    await expect(radio('Suporte em bug')).toBeFocused()
    await expect(radio('Suporte em bug')).toHaveAttribute(
      'aria-checked',
      'true'
    )
    await page.keyboard.press('ArrowRight')
    await expect(radio('Deploy de projeto')).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(radio('Deploy de projeto')).toHaveAttribute(
      'aria-checked',
      'true'
    )
    await expect(typeGroup(page).locator('[aria-checked="true"]')).toHaveCount(
      1
    )
  })

  test('a selected type continues to the details step (AC-2.3)', async ({
    page
  }) => {
    // A click anywhere on the card selects it.
    await typeGroup(page).getByText('Outro', { exact: true }).click()
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(stepHeading(page, 'Detalhes')).toBeFocused()
    await expect(
      page
        .getByRole('navigation', { name: 'Etapas do ticket' })
        .getByRole('button')
    ).toHaveText(['TipoEscolha do pedido(concluída)'])
  })
})
