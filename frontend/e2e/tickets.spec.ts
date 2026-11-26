import { type Page, expect, test } from '@playwright/test'

import { comumUser, seedAdmin } from './support/env'
import { loginAs, recordApiCalls, submitLogin } from './support/login'

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
