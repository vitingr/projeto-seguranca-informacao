import { type Locator, type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs, recordApiCalls } from './support/login'
import {
  callApi,
  createViaApi,
  pageAs,
  postNamed,
  registerComum,
  unique
} from './support/posts'
import { expectNothingInBrowserStorage } from './support/storage'

// The post wall (spec 005). Every call goes through the browser and the
// /api proxy, with the session cookie and the CSRF token of the page. The
// helpers live in support/posts.ts, shared with moderation.spec.ts (006).

const fillPost = async (scope: Locator | Page, title: string, body: string) => {
  await scope.getByLabel('Título').fill(title)
  await scope.getByLabel('Texto').fill(body)
}

test.describe('post wall as comum', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/minha-conta')
    await page.goto('/mural')
  })

  test('creates, edits and deletes a post', async ({ page }) => {
    const title = `Primeiro post ${unique()}`
    const form = page.getByRole('form', { name: 'Novo post' })
    await fillPost(form, title, 'linha 1\nlinha 2')
    await form.getByRole('button', { name: 'Publicar' }).click()

    const post = postNamed(page, title)
    await expect(post).toBeVisible()
    await expect(post.getByTestId('post-body')).toHaveText('linha 1\nlinha 2')
    await expect(post).toContainText(comumUser().login)
    await expect(form.getByLabel('Título')).toHaveValue('')

    await post.getByRole('button', { name: 'Editar' }).click()
    const edit = page.getByRole('form', { name: 'Editar post' })
    const edited = `Post revisado ${unique()}`
    await fillPost(edit, edited, 'novo texto')
    await edit.getByRole('button', { name: 'Salvar' }).click()
    const editedPost = postNamed(page, edited)
    await expect(editedPost.getByTestId('post-body')).toHaveText('novo texto')
    await expect(editedPost).toContainText('· editado')
    await expect(postNamed(page, title)).toHaveCount(0)

    await editedPost.getByRole('button', { name: 'Excluir' }).click()
    await editedPost.getByRole('button', { name: 'Confirmar exclusão' }).click()
    await expect(postNamed(page, edited)).toHaveCount(0)
    await expectNothingInBrowserStorage(page)
  })

  // 005 S-3: the content is shown as text, never as HTML or markdown.
  test('shows HTML and markdown as plain text', async ({ page }) => {
    const title = `<b>negrito</b> ${unique()}`
    const body = '<img src=x onerror="window.__xss = 1">\n**não é markdown**'
    const id = await createViaApi(page, title, body)
    try {
      await page.reload()
      const post = postNamed(page, title)
      await expect(post).toBeVisible()
      await expect(post.getByTestId('post-body')).toHaveText(body)
      await expect(post.locator('img, b, strong, script')).toHaveCount(0)
      expect(await page.evaluate(() => '__xss' in window)).toBe(false)
    } finally {
      await callApi(page, 'DELETE', `/api/posts/${id}`)
    }
  })

  // 005 S-2: zod checks the same limits before sending.
  test('client rules show field messages and send nothing', async ({
    page
  }) => {
    const calls = recordApiCalls(page)
    const form = page.getByRole('form', { name: 'Novo post' })
    await fillPost(form, '   ', 'x'.repeat(2001))
    await form.getByRole('button', { name: 'Publicar' }).click()
    await expect(form.getByText('Informe um título.')).toBeVisible()
    await expect(
      form.getByText('O texto deve ter no máximo 2000 caracteres.')
    ).toBeVisible()

    await fillPost(form, 'a'.repeat(121), '')
    await form.getByRole('button', { name: 'Publicar' }).click()
    await expect(
      form.getByText('O título deve ter no máximo 120 caracteres.')
    ).toBeVisible()
    await expect(form.getByText('Escreva o texto do post.')).toBeVisible()
    expect(calls.filter(c => c === 'POST /api/posts')).toEqual([])
  })

  test('pages through the wall, newest first', async ({ page }) => {
    const tag = unique()
    const ids: number[] = []
    try {
      for (let i = 1; i <= 21; i++) {
        ids.push(await createViaApi(page, `Paginado ${tag} ${i}`))
      }
      await page.reload()
      await expect(page.getByTestId('post')).toHaveCount(20)
      // Oldest at the top, newest at the bottom, next to the composer.
      await expect(page.getByTestId('post').last()).toContainText(
        `Paginado ${tag} 21`
      )
      await expect(postNamed(page, `Paginado ${tag} 1`)).toHaveCount(0)

      await page.getByRole('link', { name: 'Anteriores' }).click()
      await expect(page).toHaveURL('/mural?pagina=2')
      await expect(postNamed(page, `Paginado ${tag} 1`)).toBeVisible()
      await page.getByRole('link', { name: 'Mais recentes' }).click()
      await expect(page).toHaveURL('/mural')
    } finally {
      for (const id of ids) await callApi(page, 'DELETE', `/api/posts/${id}`)
    }
  })
})

// 005 S-1: only the author edits or deletes a post; the backend answers 404
// to anyone else and the post stays intact.
test('another user cannot edit or delete a post', async ({ browser }) => {
  const author = await pageAs(browser, comumUser())
  const other = await pageAs(browser, await registerComum(browser))
  const title = `Post do autor ${unique()}`
  const id = await createViaApi(author, title, 'texto original')
  try {
    await other.goto('/mural')
    const seen = postNamed(other, title)
    await expect(seen).toBeVisible()
    await expect(seen.getByRole('button')).toHaveCount(0)

    const update = await callApi(other, 'PUT', `/api/posts/${id}`, {
      body: 'invadido',
      title: 'invadido'
    })
    const remove = await callApi(other, 'DELETE', `/api/posts/${id}`)
    const missing = await callApi(other, 'DELETE', '/api/posts/999999999')
    for (const answer of [update, remove]) {
      expect(answer.status).toBe(404)
      // The same answer as a post that does not exist.
      expect((answer.body as { message: string }).message).toBe(
        (missing.body as { message: string }).message
      )
    }

    await author.goto('/mural')
    const intact = postNamed(author, title)
    await expect(intact.getByTestId('post-body')).toHaveText('texto original')
    await expect(intact.getByRole('button', { name: 'Editar' })).toBeVisible()
  } finally {
    await callApi(author, 'DELETE', `/api/posts/${id}`)
    await author.context().close()
    await other.context().close()
  }
})

// 006 D-1 changed this test: the admin now reads the wall (to moderate it)
// but still cannot create, edit or delete posts.
test('the wall needs a session, and admins only read it', async ({
  browser,
  page
}) => {
  const anonymous = await browser.newPage()
  await anonymous.goto('/mural')
  await expect(anonymous).toHaveURL('/login?next=%2Fmural')
  await anonymous.close()

  const author = await pageAs(browser, comumUser())
  const title = `Post lido pelo admin ${unique()}`
  const id = await createViaApi(author, title)
  try {
    await loginAs(page, seedAdmin(), '/admin')
    await page.goto('/mural')
    await expect(page.getByTestId('forbidden')).toHaveCount(0)
    await expect(page.getByRole('form', { name: 'Novo post' })).toHaveCount(0)
    const seen = postNamed(page, title)
    await expect(seen).toBeVisible()
    await expect(seen.getByRole('button', { name: 'Editar' })).toHaveCount(0)
    await expect(seen.getByRole('button', { name: 'Excluir' })).toHaveCount(0)

    const list = await callApi(page, 'GET', '/api/posts')
    expect(list.status).toBe(200)
    const create = await callApi(page, 'POST', '/api/posts', {
      body: 'b',
      title: 'do admin'
    })
    const update = await callApi(page, 'PUT', `/api/posts/${id}`, {
      body: 'b',
      title: 'editado pelo admin'
    })
    const remove = await callApi(page, 'DELETE', `/api/posts/${id}`)
    for (const answer of [create, update, remove]) {
      expect(answer.status).toBe(403)
    }
    await author.goto('/mural')
    await expect(postNamed(author, title).getByTestId('post-body')).toHaveText(
      'Corpo.'
    )
  } finally {
    await callApi(author, 'DELETE', `/api/posts/${id}`)
    await author.context().close()
  }
})
