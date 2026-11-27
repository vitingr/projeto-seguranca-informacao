import { type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs } from './support/login'

const nav = (page: Page) => page.getByRole('navigation', { name: 'Principal' })

const menuButton = (page: Page) =>
  nav(page).getByRole('button', { name: 'Menu' })

const visibleRole = (page: Page, roleName: string) =>
  nav(page).getByText(`Papel: ${roleName}`).filter({ visible: true })

const ROLES = [
  {
    credentials: seedAdmin,
    landing: '/administradores',
    links: ['Área do administrador', 'Cadastrar usuários'],
    name: 'admin',
    roleName: 'administrador',
    target: { label: 'Cadastrar usuários', path: '/cadastro' }
  },
  {
    credentials: comumUser,
    landing: '/usuarios',
    links: ['Minha conta'],
    name: 'comum',
    roleName: 'usuário comum',
    target: { label: 'Minha conta', path: '/usuarios' }
  }
]

for (const role of ROLES) {
  test.describe(`header at 360 px as ${role.name}`, () => {
    test.use({ viewport: { height: 800, width: 360 } })

    test.beforeEach(async ({ page }) => {
      await loginAs(page, role.credentials(), role.landing)
    })

    test('fits on one line, with "Sair" visible', async ({ page }) => {
      const controls = [
        nav(page).getByRole('link', { name: 'Início' }),
        menuButton(page),
        nav(page).getByRole('button', { name: 'Sair' })
      ]
      const boxes: { height: number; width: number; x: number; y: number }[] =
        []
      for (const control of controls) {
        await expect(control).toBeVisible()
        const box = await control.boundingBox()
        expect(box).not.toBeNull()
        boxes.push(box!)
      }

      const center = (box: { height: number; y: number }) =>
        box.y + box.height / 2
      for (const box of boxes) {
        expect(Math.abs(center(box) - center(boxes[0]))).toBeLessThanOrEqual(2)
        expect(box.x).toBeGreaterThanOrEqual(0)
        expect(box.x + box.width).toBeLessThanOrEqual(360)
      }

      const sair = boxes[2]
      expect(sair.height).toBeLessThanOrEqual(boxes[1].height)

      const overflow = await page.evaluate(() => {
        const element = document.querySelector('nav[aria-label="Principal"]')!
        return {
          nav: element.scrollWidth - element.clientWidth,
          page: document.documentElement.scrollWidth - window.innerWidth
        }
      })
      expect(overflow).toEqual({ nav: 0, page: 0 })

      await expect(nav(page).getByRole('link')).toHaveText(['Início'])
      await expect(visibleRole(page, role.roleName)).toHaveCount(0)
    })

    test('the menu lists the role links and closes on Escape', async ({
      page
    }) => {
      const button = menuButton(page)
      await expect(button).toHaveAttribute('aria-expanded', 'false')

      await button.click()
      await expect(button).toHaveAttribute('aria-expanded', 'true')
      await expect(nav(page).getByRole('link')).toHaveText([
        'Início',
        ...role.links
      ])
      await expect(visibleRole(page, role.roleName)).toHaveCount(1)

      await page.keyboard.press('Escape')
      await expect(button).toHaveAttribute('aria-expanded', 'false')
      await expect(button).toBeFocused()
      await expect(nav(page).getByRole('link')).toHaveText(['Início'])
    })

    test('a link in the menu navigates and closes it', async ({ page }) => {
      await menuButton(page).click()
      await nav(page).getByRole('link', { name: role.target.label }).click()
      await expect(page).toHaveURL(role.target.path)
      await expect(menuButton(page)).toHaveAttribute('aria-expanded', 'false')
      await expect(nav(page).getByRole('link')).toHaveText(['Início'])
    })
  })
}

test.describe('header at 1280 px', () => {
  test.use({ viewport: { height: 800, width: 1280 } })

  test('shows the links inline and no menu button', async ({ page }) => {
    await loginAs(page, seedAdmin(), '/administradores')
    await expect(menuButton(page)).toBeHidden()
    await expect(nav(page).getByRole('link')).toHaveText([
      'Início',
      'Área do administrador',
      'Cadastrar usuários'
    ])
    await expect(visibleRole(page, 'administrador')).toHaveCount(1)
  })
})
