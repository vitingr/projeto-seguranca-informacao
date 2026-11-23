import { type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs, recordApiCalls, submitLogin } from './support/login'
import {
  callApi,
  createViaApi,
  pageAs,
  postNamed,
  registerComum,
  unique
} from './support/posts'

// Moderation and the audit log (spec 006), through the browser and the /api
// proxy. A removed post is frozen, so these tests remove posts of a user
// registered for the test, never of the run's shared comum user.

interface AuditItem {
  actor: { user_id: number } | null
  event: string
  target: { id: number; type: string } | null
}

const auditItems = async (page: Page, query = '') => {
  const answer = await callApi(page, 'GET', `/api/audit-events${query}`)
  expect(answer.status).toBe(200)
  return (answer.body as { items: AuditItem[] }).items
}

/** No horizontal scrolling at the current viewport. */
const expectNoHorizontalScroll = async (page: Page) => {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - window.innerWidth
  )
  expect(overflow).toBeLessThanOrEqual(0)
}

// 006 S-6 and D-1: the admin removes a post with a reason; other comum users
// no longer see it; the author and the admin see it marked, with the
// reason; it is frozen.
test('an admin removes a post and its author sees the notice', async ({
  browser
}) => {
  const author = await pageAs(browser, await registerComum(browser))
  const other = await pageAs(browser, comumUser())
  const admin = await pageAs(browser, seedAdmin(), '/admin')
  const title = `Post a moderar ${unique()}`
  const id = await createViaApi(author, title, 'Texto que será removido.')
  try {
    await admin.goto('/mural')
    await expect(admin.getByTestId('moderation-hint')).toBeVisible()
    await expect(admin.getByRole('form', { name: 'Novo post' })).toHaveCount(0)
    const post = postNamed(admin, title)
    await expect(post.getByRole('button', { name: 'Editar' })).toHaveCount(0)
    await post.getByRole('button', { name: 'Remover' }).click()
    await post.getByLabel('Motivo').fill('  Spam repetido.\nSegunda linha.  ')
    await post.getByRole('button', { name: 'Confirmar remoção' }).click()

    const notice = 'Removido pela moderação: Spam repetido.\nSegunda linha.'
    await expect(post.getByTestId('post-removal')).toHaveText(notice)
    await expect(post).toHaveAttribute('data-removed', 'true')
    await expect(post.getByTestId('post-body')).toHaveCount(0)
    await expect(post.getByRole('button')).toHaveCount(0)

    // The admin can open the removed body, collapsed by default.
    const removedBody = post.getByTestId('post-removed-body')
    await expect(removedBody).toBeHidden()
    await post.getByText('Ver conteúdo removido').click()
    await expect(removedBody).toBeVisible()
    await expect(removedBody).toHaveText('Texto que será removido.')

    await other.goto('/mural')
    await expect(other.getByRole('heading', { name: 'Mural' })).toBeVisible()
    await expect(postNamed(other, title)).toHaveCount(0)

    await author.goto('/mural')
    const own = postNamed(author, title)
    await expect(own.getByTestId('post-removal')).toHaveText(notice)
    await expect(own.getByTestId('post-body')).toHaveCount(0)
    await expect(own.getByRole('button')).toHaveCount(0)
    // The author never sees the removed body again.
    await expect(own.getByText('Ver conteúdo removido')).toHaveCount(0)
    await expect(own.getByTestId('post-removed-body')).toHaveCount(0)
    await expect(own).not.toContainText('Texto que será removido.')

    // Frozen: the author's edit and delete, and a second removal, are 404.
    const update = await callApi(author, 'PUT', `/api/posts/${id}`, {
      body: 'b',
      title: 't'
    })
    const remove = await callApi(author, 'DELETE', `/api/posts/${id}`)
    const again = await callApi(admin, 'POST', `/api/posts/${id}/removal`, {
      reason: 'De novo.'
    })
    expect([update.status, remove.status, again.status]).toEqual([
      404, 404, 404
    ])

    // D-3: the removal wrote its event.
    const [event] = await auditItems(admin, '?type=post_removed')
    expect(event?.target).toEqual({ id, type: 'post' })
  } finally {
    await author.context().close()
    await other.context().close()
    await admin.context().close()
  }
})

test('the removal reason is checked before sending', async ({ browser }) => {
  const author = await pageAs(browser, comumUser())
  const admin = await pageAs(browser, seedAdmin(), '/admin')
  const title = `Post intacto ${unique()}`
  const id = await createViaApi(author, title)
  try {
    const calls = recordApiCalls(admin)
    await admin.goto('/mural')
    const post = postNamed(admin, title)
    await post.getByRole('button', { name: 'Remover' }).click()
    const confirm = post.getByRole('button', { name: 'Confirmar remoção' })

    await post.getByLabel('Motivo').fill('   ')
    await confirm.click()
    await expect(post.getByText('Informe o motivo da remoção.')).toBeVisible()
    await post.getByLabel('Motivo').fill('a'.repeat(501))
    await confirm.click()
    await expect(
      post.getByText('O motivo deve ter no máximo 500 caracteres.')
    ).toBeVisible()
    expect(calls.filter(c => c.endsWith('/removal'))).toEqual([])

    await post.getByRole('button', { name: 'Cancelar' }).click()
    await expect(post.getByLabel('Motivo')).toHaveCount(0)
    await expect(post.getByTestId('post-body')).toBeVisible()
  } finally {
    await callApi(author, 'DELETE', `/api/posts/${id}`)
    await author.context().close()
    await admin.context().close()
  }
})

// 006 S-3: the backend refuses comum users, whatever the page shows.
test('the audit log and the removal are admin only', async ({
  browser,
  page
}) => {
  const anonymous = await browser.newPage()
  await anonymous.goto('/admin/auditoria')
  await expect(anonymous).toHaveURL('/login?next=%2Fadmin%2Fauditoria')
  await anonymous.close()

  const author = await pageAs(browser, comumUser())
  const id = await createViaApi(author, `Post do comum ${unique()}`)
  try {
    await loginAs(page, comumUser(), '/minha-conta')
    await page.goto('/admin/auditoria')
    await expect(page.getByTestId('forbidden')).toBeVisible()
    await expect(page.getByTestId('audit-row')).toHaveCount(0)

    const list = await callApi(page, 'GET', '/api/audit-events')
    const removal = await callApi(page, 'POST', `/api/posts/${id}/removal`, {
      reason: 'Tentativa do comum.'
    })
    expect([list.status, removal.status]).toEqual([403, 403])
    await page.goto('/mural')
    await expect(page.getByRole('button', { name: 'Remover' })).toHaveCount(0)
  } finally {
    await callApi(author, 'DELETE', `/api/posts/${id}`)
    await author.context().close()
  }
})

// 006 S-2 and D-8: the page lists and filters the events, and no event
// holds a password, a token, a cookie or a typed wrong login.
test('the audit page lists and filters events with no secret', async ({
  browser
}) => {
  const typedLogin = `Senha-Digitada-No-Login-${unique()}`
  const wrongPassword = `senha-errada-${unique()}`
  const attempts = await browser.newPage()
  for (const login of [typedLogin, comumUser().login]) {
    await attempts.goto('/login')
    await submitLogin(attempts, { login, password: wrongPassword })
    await expect(attempts.getByText('Login ou senha incorretos.')).toBeVisible()
  }
  await attempts.close()

  const admin = await pageAs(browser, seedAdmin(), '/admin')
  try {
    await admin.goto('/admin/auditoria')
    await expect(admin.getByTestId('audit-row').first()).toBeVisible()
    await admin.getByLabel('Tipo de evento').selectOption('login_failed')
    await admin.getByRole('button', { name: 'Filtrar' }).click()
    await expect(admin).toHaveURL('/admin/auditoria?tipo=login_failed')
    const rows = admin.getByTestId('audit-row')
    await expect(rows.first()).toBeVisible()
    for (const event of await rows.evaluateAll(els =>
      els.map(el => el.getAttribute('data-event'))
    )) {
      expect(event).toBe('login_failed')
    }
    // The newest failed login named the existing comum user as its target.
    await expect(rows.first().getByTestId('audit-target')).toContainText(
      comumUser().login
    )
    await expect(rows.first().getByTestId('audit-actor')).toContainText('—')

    const token = (
      (await callApi(admin, 'GET', '/api/csrf')).body as { csrf_token: string }
    ).csrf_token
    const cookies = (await admin.context().cookies()).map(c => c.value)
    const secrets = [
      typedLogin,
      typedLogin.toLowerCase(),
      wrongPassword,
      seedAdmin().password,
      comumUser().password,
      token,
      ...cookies
    ]
    const listed = JSON.stringify(await auditItems(admin))
    const html = await admin.content()
    for (const secret of secrets) {
      expect(listed).not.toContain(secret)
      expect(html).not.toContain(secret)
    }
  } finally {
    await admin.context().close()
  }
})

// The UI rules: every screen works from 360px wide.
test('the wall and the audit page fit a 360px screen', async ({ browser }) => {
  const admin = await pageAs(browser, seedAdmin(), '/admin')
  try {
    for (const width of [360, 768, 1280]) {
      await admin.setViewportSize({ height: 800, width })
      for (const path of ['/mural', '/admin/auditoria']) {
        await admin.goto(path)
        await expect(admin.getByRole('heading', { level: 1 })).toBeVisible()
        await expectNoHorizontalScroll(admin)
      }
    }
  } finally {
    await admin.context().close()
  }
})
