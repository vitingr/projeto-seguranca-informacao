import { randomBytes } from 'node:crypto'

import { type Browser, type Page, expect, test } from '@playwright/test'

import { comumUser, generateComumCredentials, seedAdmin } from './support/env'
import { loginAs, recordApiCalls, submitLogin } from './support/login'
import { expectNothingInBrowserStorage } from './support/storage'

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

// The details step (task T9.6). No ticket is sent here, so the shared comum
// user stays far from the rate limit.

const goToDetails = async (page: Page, typeLabel: string) => {
  await typeGroup(page).getByText(typeLabel, { exact: true }).click()
  await page.getByRole('button', { name: 'Continuar' }).click()
  await expect(stepHeading(page, 'Detalhes')).toBeFocused()
}

const choose = async (page: Page, label: string, option: string) => {
  await page.getByRole('combobox', { name: label }).click()
  await page.getByRole('option', { name: new RegExp(`^${option}`) }).click()
}

const VALID: Record<string, (page: Page) => Promise<void>> = {
  'Deploy de projeto': async page => {
    await page.getByLabel('Nome do projeto').fill('api-pagamentos')
    await page.getByLabel('Repositório').fill('https://github.com/org/api')
    await choose(page, 'Ambiente', 'Homologação')
  },
  Outro: async page => {
    await page.getByLabel('Título').fill('Liberar porta no firewall')
    await page
      .getByLabel('Descrição')
      .fill('Preciso da porta 8443 aberta no servidor de testes.')
  },
  'Solicitar acesso': async page => {
    await page.getByLabel('Sistema ou recurso').fill('Grafana')
    await choose(page, 'Nível de acesso', 'Leitura')
    await page
      .getByLabel('Justificativa')
      .fill('Preciso consultar os painéis de uso do mês.')
  },
  'Suporte em bug': async page => {
    await page.getByLabel('Sistema afetado').fill('Portal do aluno')
    await choose(page, 'Severidade', 'Alta')
    await page
      .getByLabel('Descrição')
      .fill('Ao salvar o formulário, a página mostra erro 500.')
    await page
      .getByLabel('Passos para reproduzir')
      .fill('1. Abra o portal\n2. Clique em Salvar')
  }
}

test.describe('details step', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
    await page.goto(BUILDER)
  })

  test('shows the fields of the type, marked, with the session note (AC-2.3, AC-3.1, AC-3.7)', async ({
    page
  }) => {
    await goToDetails(page, 'Deploy de projeto')
    const card = page.locator('form')
    await expect(card.getByLabel('Nome do projeto')).toHaveValue('')
    await expect(card.getByLabel('Repositório')).toHaveValue('')
    await expect(
      page.getByRole('combobox', { name: 'Ambiente' })
    ).toHaveAttribute('aria-required', 'true')
    // Required: a red asterisk, read as "obrigatório"; optional: "(opcional)".
    await expect(
      page.locator('label', { hasText: 'Nome do projeto' })
    ).toHaveText('Nome do projeto*(obrigatório)')
    await expect(page.locator('label', { hasText: 'Observações' })).toHaveText(
      'Observações(opcional)'
    )
    await expect(card.getByTestId(/-observacoes-counter$/)).toHaveText(
      '0 / 2000'
    )
    await expect(
      page.getByText(
        'Sua sessão expira após um período sem atividade. Envie o ticket antes de sair da página.'
      )
    ).toBeVisible()
  })

  test('an empty form stays, explains each field and sends nothing (AC-3.2, AC-3.5, S-2)', async ({
    page
  }) => {
    const calls = recordApiCalls(page)
    await goToDetails(page, 'Suporte em bug')
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(stepHeading(page, 'Detalhes')).toBeVisible()
    // The first invalid field, in screen order, gets the focus.
    await expect(page.getByLabel('Sistema afetado')).toBeFocused()
    await expect(page.getByText('Preencha este campo.')).toHaveCount(3)
    await expect(page.getByText('Escolha uma opção.')).toHaveCount(1)
    await expect(page.getByLabel('Descrição')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
    expect(calls.filter(call => call.startsWith('POST /api/tickets'))).toEqual(
      []
    )
  })

  test('format and length errors say what is expected (AC-3.3, AC-3.4)', async ({
    page
  }) => {
    const calls = recordApiCalls(page)
    await goToDetails(page, 'Deploy de projeto')
    await VALID['Deploy de projeto'](page)
    await page.getByLabel('Repositório').fill('git@github.com:org/api.git')
    await page.getByLabel('Nome do projeto').fill('a'.repeat(121))
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(page.getByLabel('Nome do projeto')).toBeFocused()
    await expect(page.getByText('Use no máximo 120 caracteres.')).toBeVisible()
    await expect(
      page.getByText(
        'Informe um endereço https:// completo, como https://github.com/org/projeto.'
      )
    ).toBeVisible()
    await expect(stepHeading(page, 'Detalhes')).toBeVisible()
    expect(calls.filter(call => call.startsWith('POST /api/tickets'))).toEqual(
      []
    )
  })

  test('a field left empty explains itself when the user leaves it (AC-3.2)', async ({
    page
  }) => {
    await goToDetails(page, 'Outro')
    await page.getByLabel('Título').focus()
    await page.keyboard.press('Tab')
    await expect(page.getByText('Preencha este campo.')).toHaveCount(1)
  })

  test('the counter and the preview follow what is typed', async ({ page }) => {
    await goToDetails(page, 'Outro')
    await page.getByLabel('Título').fill('Liberar porta')
    // Code points, as the server counts them: the emoji is one.
    await page.getByLabel('Descrição').fill('ação 🚀')
    await expect(page.getByTestId(/-descricao-counter$/)).toHaveText('6 / 2000')
    const preview = page.getByRole('region', { name: 'Prévia' })
    await expect(preview).toContainText('Liberar porta')
    await expect(preview).toContainText('Outro')
  })

  test('the choice menu shows descriptions, a check and risky options in red (Q-17)', async ({
    page
  }) => {
    await goToDetails(page, 'Suporte em bug')
    await page.getByRole('combobox', { name: 'Severidade' }).click()
    const critica = page.getByRole('option', { name: /^Crítica/ })
    await expect(critica).toContainText('Sistema fora do ar ou dados em risco.')
    await critica.click()
    await expect(page.getByRole('combobox', { name: 'Severidade' })).toHaveText(
      'Crítica'
    )
    await expect(
      page.getByRole('region', { name: 'Prévia' }).getByText('Crítica')
    ).toBeVisible()
    await page.getByRole('combobox', { name: 'Severidade' }).click()
    await expect(
      page.getByRole('option', { name: /^Crítica/ })
    ).toHaveAttribute('aria-selected', 'true')
  })

  for (const typeLabel of Object.keys(VALID)) {
    test(`a valid "${typeLabel}" form continues to the review (AC-3.6)`, async ({
      page
    }) => {
      const calls = recordApiCalls(page)
      await goToDetails(page, typeLabel)
      await VALID[typeLabel](page)
      await page.getByRole('button', { name: 'Continuar' }).click()
      await expect(stepHeading(page, 'Revisão')).toBeFocused()
      // Back to the form, every value is kept (AC-4.2).
      await page.getByRole('button', { name: 'Editar' }).click()
      await expect(stepHeading(page, 'Detalhes')).toBeFocused()
      await expect(
        page.locator('form input[type="text"]').first()
      ).not.toHaveValue('')
      expect(
        calls.filter(call => call.startsWith('POST /api/tickets'))
      ).toEqual([])
    })
  }

  test('changing the type with data asks inline first (AC-2.4)', async ({
    page
  }) => {
    await goToDetails(page, 'Outro')
    await page.getByLabel('Título').fill('Liberar porta')
    await page.getByRole('button', { name: 'Voltar' }).click()
    await expect(stepHeading(page, 'Qual é o tipo do pedido?')).toBeFocused()

    await typeGroup(page).getByText('Suporte em bug', { exact: true }).click()
    const confirm = page.getByTestId('type-change')
    await expect(confirm).toHaveText(
      /Trocar para "Suporte em bug" descarta os campos preenchidos de "Outro"\./
    )
    await expect(
      typeGroup(page).getByRole('radio', { name: 'Outro' })
    ).toHaveAttribute('aria-checked', 'true')

    // Keep: nothing is lost.
    await confirm.getByRole('button', { name: 'Manter o tipo atual' }).click()
    await expect(confirm).toHaveCount(0)
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(page.getByLabel('Título')).toHaveValue('Liberar porta')

    // Change: the new type starts empty.
    await page.getByRole('button', { name: 'Voltar' }).click()
    await typeGroup(page).getByText('Suporte em bug', { exact: true }).click()
    await page
      .getByTestId('type-change')
      .getByRole('button', { name: 'Trocar e descartar' })
      .click()
    await expect(
      typeGroup(page).getByRole('radio', { name: 'Suporte em bug' })
    ).toHaveAttribute('aria-checked', 'true')
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(page.getByLabel('Sistema afetado')).toHaveValue('')
    await expect(page.getByLabel('Título')).toHaveCount(0)
  })

  test('changing the type with no data asks nothing', async ({ page }) => {
    await goToDetails(page, 'Outro')
    await page.getByRole('button', { name: 'Voltar' }).click()
    await typeGroup(page).getByText('Suporte em bug', { exact: true }).click()
    await expect(page.getByTestId('type-change')).toHaveCount(0)
    await expect(
      typeGroup(page).getByRole('radio', { name: 'Suporte em bug' })
    ).toHaveAttribute('aria-checked', 'true')
  })
})

// The review step (task T9.7).

const reviewCard = (page: Page) => page.getByRole('region', { name: 'Revisão' })

// The value next to a label; "Título" is both the ticket title and, for
// "Outro", a field, so `.first()` picks the "Pedido" group.
const reviewValue = (page: Page, label: string) =>
  reviewCard(page)
    .locator('dt', { hasText: new RegExp(`^${label}$`) })
    .locator('xpath=following-sibling::dd[1]')

test.describe('review step', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
    await page.goto(BUILDER)
  })

  test('shows the type, the title and every field as it will be sent (AC-4.1)', async ({
    page
  }) => {
    await goToDetails(page, 'Deploy de projeto')
    await page.getByLabel('Nome do projeto').fill('  api-pagamentos  ')
    await page.getByLabel('Repositório').fill('https://github.com/org/api')
    await choose(page, 'Ambiente', 'Produção')
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(stepHeading(page, 'Revisão')).toBeFocused()

    await expect(reviewValue(page, 'Tipo')).toHaveText('Deploy de projeto')
    await expect(reviewValue(page, 'Título')).toHaveText(
      'Deploy de api-pagamentos em Produção'
    )
    // Trimmed, as the server stores it (BR-4).
    await expect(reviewValue(page, 'Nome do projeto')).toHaveText(
      'api-pagamentos'
    )
    await expect(reviewValue(page, 'Ambiente')).toHaveText('Produção')
    await expect(reviewValue(page, 'Observações')).toHaveText('Não informado')
  })

  test('shows HTML and markdown literally and runs nothing (AC-4.3, S-8)', async ({
    page
  }) => {
    let dialogs = 0
    page.on('dialog', async dialog => {
      dialogs += 1
      await dialog.dismiss()
    })
    const title = '<img src=x onerror=alert(1)>'
    const description = '<script>alert(1)</script>\n**negrito**'
    await goToDetails(page, 'Outro')
    await page.getByLabel('Título').fill(title)
    await page.getByLabel('Descrição').fill(description)
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(stepHeading(page, 'Revisão')).toBeFocused()

    await expect(reviewValue(page, 'Título').first()).toHaveText(title)
    await expect(reviewValue(page, 'Descrição')).toHaveText(description)
    await expect(reviewCard(page).locator('img, script, strong')).toHaveCount(0)
    // The preview shows the same title as text too.
    await expect(
      page.getByRole('region', { name: 'Prévia' }).getByText(title)
    ).toBeVisible()
    await expect(page.locator('img[src="x"]')).toHaveCount(0)
    expect(dialogs).toBe(0)
  })

  test('"Editar" goes back to the form with every value kept (AC-4.2)', async ({
    page
  }) => {
    await goToDetails(page, 'Suporte em bug')
    await VALID['Suporte em bug'](page)
    await page.getByRole('button', { name: 'Continuar' }).click()
    await expect(reviewValue(page, 'Severidade')).toHaveText('Alta')
    await page.getByRole('button', { name: 'Editar' }).click()
    await expect(stepHeading(page, 'Detalhes')).toBeFocused()
    await expect(page.getByLabel('Sistema afetado')).toHaveValue(
      'Portal do aluno'
    )
    await expect(page.getByRole('combobox', { name: 'Severidade' })).toHaveText(
      'Alta'
    )
    await expect(page.getByLabel('Passos para reproduzir')).toHaveValue(
      '1. Abra o portal\n2. Clique em Salvar'
    )
  })
})

// Sending and the confirmation (task T9.8). Real tickets are created by a
// user of their own, registered here, so the shared comum user never gets
// near the limit of 10 per minute (S-12, plan risk table). The error
// answers are simulated in the browser and create nothing.

const registerComum = async (browser: Browser) => {
  const user = generateComumCredentials(randomBytes)
  const context = await browser.newContext()
  const admin = await context.newPage()
  await loginAs(admin, seedAdmin(), '/administradores')
  const status = await admin.evaluate(
    async ([login, senha]) => {
      const token = (await (await fetch('/api/csrf')).json()).csrf_token
      const res = await fetch('/api/cadastro', {
        body: JSON.stringify({ login, papel: 'comum', senha }),
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token },
        method: 'POST'
      })
      return res.status
    },
    [user.login, user.password]
  )
  expect(status).toBe(201)
  await context.close()
  return user
}

const TITLES: Record<string, string> = {
  'Deploy de projeto': 'Deploy de api-pagamentos em Homologação',
  Outro: 'Liberar porta no firewall',
  'Solicitar acesso': 'Acesso a Grafana',
  'Suporte em bug': 'Bug em Portal do aluno'
}

const toReview = async (page: Page, typeLabel: string) => {
  await page.goto(BUILDER)
  await goToDetails(page, typeLabel)
  await VALID[typeLabel](page)
  await page.getByRole('button', { name: 'Continuar' }).click()
  await expect(stepHeading(page, 'Revisão')).toBeFocused()
}

const posts = (calls: string[]) =>
  calls.filter(call => call === 'POST /api/tickets')

// The send progress (task T9.8a, AC-5.5 to AC-5.10).

const progress = (page: Page) => page.getByTestId('send-progress')

const stepStates = (page: Page) =>
  progress(page)
    .locator('li[data-state]')
    .evaluateAll(items =>
      items.map(item => [
        item.firstChild?.nextSibling?.textContent?.split(' (')[0]?.trim(),
        item.getAttribute('data-state')
      ])
    )

const expectStoppedProgress = async (page: Page) => {
  await expect(progress(page)).toHaveAttribute('data-phase', 'failed')
  expect(await stepStates(page)).toEqual([
    ['Validando campos', 'done'],
    ['Registrando ticket', 'failed']
  ])
  await expect(progress(page)).not.toContainText('Preparando confirmação')
  await expect(progress(page)).toContainText('Envio interrompido')
}

const CREATED = {
  body: JSON.stringify({
    created_at: '2026-11-25T14:30:00Z',
    id: 9001,
    status: 'aberto',
    title: 'Liberar porta no firewall',
    type: 'outro',
    updated_at: '2026-11-25T14:30:00Z'
  }),
  contentType: 'application/json',
  status: 201
}

/** Holds the POST until `release()`, then answers a fake 201. */
const holdCreation = async (page: Page) => {
  let release = () => {}
  const gate = new Promise<void>(done => {
    release = done
  })
  const seen: number[] = []
  await page.route('**/api/tickets', async route => {
    seen.push(Date.now())
    await gate
    await route.fulfill(CREATED)
  })
  return { release, seen }
}

test.describe('sending a ticket', () => {
  let sender: { login: string; password: string }

  test.beforeAll(async ({ browser }) => {
    sender = await registerComum(browser)
  })

  test.beforeEach(async ({ page }) => {
    await loginAs(page, sender, '/usuarios')
  })

  for (const typeLabel of Object.keys(VALID)) {
    test(`a "${typeLabel}" ticket reaches the confirmation (AC-5.2)`, async ({
      page
    }) => {
      await toReview(page, typeLabel)
      await page.getByRole('button', { name: 'Enviar ticket' }).click()

      await expect(stepHeading(page, 'Ticket criado')).toBeFocused()
      const card = page.getByRole('region', { name: 'Ticket criado' })
      const value = (label: string) =>
        card
          .locator('dt', { hasText: new RegExp(`^${label}$`) })
          .locator('xpath=following-sibling::dd[1]')
      await expect(value('Número')).toHaveText(/^#\d+$/)
      await expect(value('Tipo')).toHaveText(typeLabel)
      await expect(value('Título')).toHaveText(TITLES[typeLabel])
      await expect(value('Status')).toHaveText('Aberto')
      await expect(value('Criado em')).toHaveText(
        /^\d{2}\/\d{2}\/\d{4} às \d{2}:\d{2}$/
      )
      await expect(page.getByRole('status').last()).toHaveText(
        /^Ticket #\d+ criado\.$/
      )
      // No step can be revisited after sending.
      await expect(
        page
          .getByRole('navigation', { name: 'Etapas do ticket' })
          .getByRole('button')
      ).toHaveCount(0)
    })
  }

  test('a double click sends one request (AC-5.1)', async ({ page }) => {
    const calls = recordApiCalls(page)
    await toReview(page, 'Outro')
    await page.getByRole('button', { name: 'Enviar ticket' }).dblclick()
    await expect(stepHeading(page, 'Ticket criado')).toBeVisible()
    expect(posts(calls)).toHaveLength(1)
  })

  test('nothing typed stays in storage or in the URL (S-10), and "Criar outro ticket" starts over (AC-5.4)', async ({
    page
  }) => {
    await toReview(page, 'Suporte em bug')
    await page.getByRole('button', { name: 'Enviar ticket' }).click()
    await expect(stepHeading(page, 'Ticket criado')).toBeVisible()
    await expectNothingInBrowserStorage(page)
    expect(new URL(page.url()).pathname).toBe(BUILDER)
    expect(page.url()).not.toContain('Portal')

    await page.getByRole('button', { name: 'Criar outro ticket' }).click()
    await expect(stepHeading(page, 'Qual é o tipo do pedido?')).toBeFocused()
    await expect(typeGroup(page).locator('[aria-checked="true"]')).toHaveCount(
      0
    )
    await goToDetails(page, 'Suporte em bug')
    await expect(page.getByLabel('Sistema afetado')).toHaveValue('')
  })

  test('"Voltar para usuários" goes to /usuarios', async ({ page }) => {
    await toReview(page, 'Outro')
    await page.getByRole('button', { name: 'Enviar ticket' }).click()
    await page.getByRole('link', { name: 'Voltar para usuários' }).click()
    await expect(page).toHaveURL('/usuarios')
  })
})

test.describe('errors when sending (US-6)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
  })

  const answer = (status: number, error: string) => ({
    body: JSON.stringify({ error, message: 'x', request_id: 'req-e2e-1' }),
    contentType: 'application/json',
    status
  })

  for (const [name, reply, message] of [
    [
      'invalid data (422, AC-6.1)',
      answer(422, 'unprocessable_entity'),
      'Dados inválidos. Revise os campos e tente novamente.'
    ],
    [
      'too many requests (429, AC-6.7)',
      answer(429, 'too_many_requests'),
      'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.'
    ],
    [
      'lost permission (403, AC-6.5)',
      answer(403, 'forbidden'),
      'Você não tem permissão para realizar esta ação.'
    ],
    [
      'server error (500, AC-6.4)',
      answer(500, 'internal'),
      'Ocorreu um erro inesperado. Tente novamente mais tarde. (código: req-e2e-1)'
    ]
  ] as const) {
    test(`${name} keeps the review and the data`, async ({ page }) => {
      await page.route('**/api/tickets', route => route.fulfill(reply))
      await toReview(page, 'Outro')
      await page.getByRole('button', { name: 'Enviar ticket' }).click()

      await expect(page.getByTestId('send-error')).toHaveText(message)
      await expect(page.getByTestId('send-error')).toHaveAttribute(
        'role',
        'status'
      )
      // The progress stops on "Registrando ticket" (AC-5.7).
      await expectStoppedProgress(page)
      await expect(stepHeading(page, 'Revisão')).toBeVisible()
      await expect(
        page.getByRole('button', { name: 'Enviar ticket' })
      ).toBeEnabled()
      // The data is kept: back on the form, every value is there.
      await page.getByRole('button', { name: 'Editar' }).click()
      await expect(page.getByLabel('Título')).toHaveValue(
        'Liberar porta no firewall'
      )
    })
  }

  test('a network failure keeps the data and allows sending again (AC-6.3)', async ({
    page
  }) => {
    let fail = true
    await page.route('**/api/tickets', route =>
      fail
        ? route.abort('failed')
        : route.fulfill({
            body: JSON.stringify({
              created_at: '2026-11-25T14:30:00Z',
              id: 9001,
              status: 'aberto',
              title: 'Liberar porta no firewall',
              type: 'outro',
              updated_at: '2026-11-25T14:30:00Z'
            }),
            contentType: 'application/json',
            status: 201
          })
    )
    await toReview(page, 'Outro')
    await page.getByRole('button', { name: 'Enviar ticket' }).click()
    await expect(page.getByTestId('send-error')).toHaveText(
      'Não foi possível conectar ao servidor. Tente novamente.'
    )
    await expectStoppedProgress(page)
    fail = false
    await page.getByRole('button', { name: 'Enviar ticket' }).click()
    await expect(stepHeading(page, 'Ticket criado')).toBeVisible()
    await expect(page.getByTestId('send-error')).toHaveCount(0)
  })

  test('the 429 is not retried by the page (AC-6.7)', async ({ page }) => {
    const calls = recordApiCalls(page)
    await page.route('**/api/tickets', route =>
      route.fulfill(answer(429, 'too_many_requests'))
    )
    await toReview(page, 'Outro')
    await page.getByRole('button', { name: 'Enviar ticket' }).click()
    await expect(page.getByTestId('send-error')).toBeVisible()
    expect(posts(calls)).toHaveLength(1)
  })

  test('an expired session goes to the login page (AC-6.2)', async ({
    page
  }) => {
    // A real expiry: the session cookie is gone when the user sends.
    await toReview(page, 'Outro')
    await page.context().clearCookies()
    await page.getByRole('button', { name: 'Enviar ticket' }).click()
    await expect(page).toHaveURL(
      `/login?next=${encodeURIComponent(BUILDER)}&sessao=expirada`
    )
    await expect(
      page.getByText('Sua sessão expirou. Entre novamente.')
    ).toBeVisible()
  })
})

test('an admin cannot create a ticket, even calling the API (S-6)', async ({
  page
}) => {
  await loginAs(page, seedAdmin(), '/administradores')
  const status = await page.evaluate(async () => {
    const token = (await (await fetch('/api/csrf')).json()).csrf_token
    const res = await fetch('/api/tickets', {
      body: JSON.stringify({
        dados: { descricao: 'Teste de permissão.', titulo: 'Teste' },
        tipo: 'outro'
      }),
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token },
      method: 'POST'
    })
    return res.status
  })
  expect(status).toBe(403)
})

// The answers here are simulated, so nothing is created.
test.describe('send progress', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, comumUser(), '/usuarios')
  })

  test('the request starts on the click and the steps follow the real events (AC-5.5, AC-5.6, AC-5.8, AC-5.10)', async ({
    page
  }) => {
    const hold = await holdCreation(page)
    await toReview(page, 'Outro')
    const clicked = Date.now()
    await page.getByRole('button', { name: 'Enviar ticket' }).click()

    // The POST left at once, before any answer.
    await expect.poll(() => hold.seen.length).toBe(1)
    expect(hold.seen[0] - clicked).toBeLessThan(1000)

    // While pending: fields checked, "Registrando ticket" in progress, no
    // confirmation step and no confirmation.
    await expect(progress(page)).toHaveAttribute('data-phase', 'registering')
    expect(await stepStates(page)).toEqual([
      ['Validando campos', 'done'],
      ['Registrando ticket', 'active']
    ])
    await expect(progress(page)).not.toContainText('Preparando confirmação')
    await expect(stepHeading(page, 'Ticket criado')).toHaveCount(0)
    await expect(progress(page)).toContainText('Enviando ticket')
    await expect(page.getByRole('status').last()).toHaveText('Enviando ticket.')
    // Chips with the type and the field names, never the values.
    const chips = progress(page).locator('ul li')
    await expect(chips).toHaveText(['Outro', 'Título', 'Descrição'])
    await expect(progress(page)).not.toContainText('Liberar porta')

    // The answer arrives: the confirmation shows with no fixed wait.
    const released = Date.now()
    hold.release()
    await expect(stepHeading(page, 'Ticket criado')).toBeVisible()
    expect(Date.now() - released).toBeLessThan(1500)
    await expect(progress(page)).toHaveAttribute('data-phase', 'done')
    expect(await stepStates(page)).toEqual([
      ['Validando campos', 'done'],
      ['Registrando ticket', 'done'],
      ['Preparando confirmação', 'done']
    ])
    await expect(progress(page)).toContainText('Envio concluído')
    await expect(progress(page)).toContainText(/Enviado em \d+,\d s/)
    await expect(page.getByRole('status').last()).toHaveText(
      'Ticket #9001 criado.'
    )
  })

  test.describe('with reduced motion (AC-5.9)', () => {
    test.use({ reducedMotion: 'reduce' })

    const animations = (page: Page) =>
      progress(page).evaluate(
        root =>
          [root, ...root.querySelectorAll('*')]
            .map(element => getComputedStyle(element))
            .filter(
              style =>
                style.animationName !== 'none' ||
                (style.filter !== 'none' && style.filter !== '')
            ).length
      )

    test('shows the same states with nothing moving', async ({ page }) => {
      const hold = await holdCreation(page)
      await toReview(page, 'Outro')
      await page.getByRole('button', { name: 'Enviar ticket' }).click()

      await expect(progress(page)).toHaveAttribute('data-phase', 'registering')
      expect(await stepStates(page)).toEqual([
        ['Validando campos', 'done'],
        ['Registrando ticket', 'active']
      ])
      expect(await animations(page)).toBe(0)

      hold.release()
      await expect(stepHeading(page, 'Ticket criado')).toBeVisible()
      await expect(progress(page)).toHaveAttribute('data-phase', 'done')
      expect(await animations(page)).toBe(0)
    })
  })
})
