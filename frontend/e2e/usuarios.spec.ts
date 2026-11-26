import { type Page, expect, test } from '@playwright/test'

import { avatarInitials } from '../src/domain/avatar'
import { comumUser, seedAdmin } from './support/env'
import { loginAs } from './support/login'

// The profile on /usuarios (spec US-7, task T9.12).

const COMUM_TEXT = 'Página de acesso restrito aos usuários comuns'
const FORBIDDEN = 'Você não tem permissão para acessar esta página.'

const identity = (page: Page) => page.getByTestId('profile-identity')

const rowValue = (page: Page, label: string) =>
  page
    .getByRole('region', { name: 'Perfil' })
    .locator('dt', { hasText: new RegExp(`^${label}$`) })
    .locator('xpath=following-sibling::dd[1]')

test.describe('profile on /usuarios as comum', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
  })

  test('keeps the exact text, the area line and "Novo ticket" (AC-7.1)', async ({
    page
  }) => {
    await expect(
      page.getByRole('heading', { level: 1, name: 'Minha conta' })
    ).toBeVisible()
    await expect(page.getByText(COMUM_TEXT, { exact: true })).toBeVisible()
    await expect(page.getByTestId('area')).toHaveText(
      'Área liberada pelo servidor: usuarios'
    )
    await page.getByRole('link', { name: 'Novo ticket' }).click()
    await expect(page).toHaveURL('/usuarios/tickets/novo')
  })

  test('shows the avatar of the login, the login, the role and "Membro desde" (AC-7.2, AC-7.3)', async ({
    page
  }) => {
    const { login } = comumUser()
    const avatar = identity(page).locator('[data-avatar-tint]')
    await expect(avatar).toHaveAttribute('aria-hidden', 'true')
    await expect(avatar).toHaveText(avatarInitials(login))
    await expect(avatar.locator('svg rect').first()).toBeAttached()
    // The login is the text read next to the decorative avatar.
    await expect(identity(page).getByText(login, { exact: true })).toBeVisible()
    await expect(identity(page)).toContainText('Usuário comum')

    await expect(rowValue(page, 'Login')).toHaveText(login)
    await expect(rowValue(page, 'Papel')).toHaveText('Usuário comum')
    await expect(rowValue(page, 'Membro desde')).toHaveText(
      /^\d{2}\/\d{2}\/\d{4}$/
    )
    await expect(page.locator('main img, main image')).toHaveCount(0)
  })

  test('states the session and rate facts (AC-7.4)', async ({ page }) => {
    const card = page.getByRole('region', { name: 'Sessão e limites' })
    await expect(card).toContainText('período sem atividade')
    await expect(card).toContainText('até 10 tickets por minuto')
  })

  test('loads nothing from another origin (AC-7.2)', async ({
    baseURL,
    page
  }) => {
    const origins = new Set<string>()
    page.on('request', request => {
      origins.add(new URL(request.url()).origin)
    })
    await page.reload()
    await expect(identity(page).locator('[data-avatar-tint]')).toBeVisible()
    await page.waitForLoadState('networkidle')
    expect([...origins]).toEqual([new URL(baseURL ?? '').origin])
  })
})

test('an admin still gets the forbidden screen on /usuarios (AC-7.5)', async ({
  page
}) => {
  await loginAs(page, seedAdmin(), '/administradores')
  await page.goto('/usuarios')
  await expect(page.getByTestId('forbidden')).toHaveText(FORBIDDEN)
  await expect(page.getByTestId('profile-identity')).toHaveCount(0)
  await expect(page.getByTestId('area')).toHaveCount(0)
})
