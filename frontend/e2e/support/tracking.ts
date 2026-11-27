import { execFileSync } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import path from 'node:path'

import { type Browser, type Page, expect } from '@playwright/test'

import { generateComumCredentials, requireEnv, seedAdmin } from './env'
import { type Credentials, loginAs } from './login'

const backendDir = path.join(__dirname, '..', '..', '..', 'backend')

export const registerComum = async (browser: Browser): Promise<Credentials> => {
  const user = generateComumCredentials(randomBytes)
  await asUser(browser, seedAdmin(), '/administradores', async admin => {
    const status = await admin.evaluate(
      async ([login, senha]) => {
        const token = (await (await fetch('/api/csrf')).json()).csrf_token
        const res = await fetch('/api/cadastro', {
          body: JSON.stringify({ login, papel: 'comum', senha }),
          headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': token
          },
          method: 'POST'
        })
        return res.status
      },
      [user.login, user.password]
    )
    expect(status).toBe(201)
  })
  return user
}

export const asUser = async <T>(
  browser: Browser,
  user: Credentials,
  lands: string,
  work: (page: Page) => Promise<T>
): Promise<T> => {
  const context = await browser.newContext()
  try {
    const page = await context.newPage()
    await loginAs(page, user, lands)
    return await work(page)
  } finally {
    await context.close()
  }
}

const write = (page: Page, method: string, url: string, body: unknown) =>
  page.evaluate(
    async ([method, url, body]) => {
      const token = (await (await fetch('/api/csrf')).json()).csrf_token
      const res = await fetch(url, {
        body: JSON.stringify(body),
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': token },
        method
      })
      return { body: await res.json(), status: res.status }
    },
    [method, url, body] as const
  )

export type TicketBody =
  | {
      dados: {
        descricao: string
        passos: string
        severidade: string
        sistema: string
      }
      tipo: 'bug'
    }
  | {
      dados: {
        ambiente: string
        observacoes?: string
        projeto: string
        repositorio: string
      }
      tipo: 'deploy'
    }
  | {
      dados: { justificativa: string; nivel_acesso: string; sistema: string }
      tipo: 'acesso'
    }
  | { dados: { descricao: string; titulo: string }; tipo: 'outro' }

export const createTicket = async (page: Page, ticket: TicketBody) => {
  const { body, status } = await write(page, 'POST', '/api/tickets', ticket)
  expect(status).toBe(201)
  return body.id as number
}

export const reply = async (
  page: Page,
  scope: 'own' | 'queue',
  id: number,
  texto: string
) => {
  const base = scope === 'own' ? '/api/tickets' : '/api/administradores/tickets'
  const { status } = await write(page, 'POST', `${base}/${id}/mensagens`, {
    texto
  })
  expect(status).toBe(201)
}

export const changeStatus = async (
  page: Page,
  id: number,
  de: string,
  para: string
) => {
  const { status } = await write(
    page,
    'PATCH',
    `/api/administradores/tickets/${id}/status`,
    { de, para }
  )
  expect(status).toBe(200)
}

// Values are psql variables, never concatenated into the SQL
export const insertOtherTickets = (login: string, count: number) => {
  const output = execFileSync(
    'docker',
    [
      'compose',
      'exec',
      '-T',
      'db',
      'psql',
      '-X',
      '-q',
      '-t',
      '-v',
      'ON_ERROR_STOP=1',
      '-U',
      requireEnv('POSTGRES_USER'),
      '-d',
      requireEnv('POSTGRES_DB'),
      '-v',
      `login=${login}`,
      '-v',
      `count=${count}`
    ],
    {
      cwd: backendDir,
      encoding: 'utf8',
      input: `WITH made AS (
  INSERT INTO tickets (author_id, type, title, description, created_at, updated_at)
  SELECT u.id, 'outro', 'Pedido ' || g, 'Descrição do pedido número ' || g,
         now() - make_interval(mins => (:count - g + 1)),
         now() - make_interval(mins => (:count - g + 1))
  FROM users u, generate_series(1, :count) AS g
  WHERE u.login = :'login'
  RETURNING id
)
SELECT count(*) FROM made;
`
    }
  )
  if (Number(output.trim()) !== count) {
    throw new Error('insertOtherTickets: tickets not created')
  }
}
