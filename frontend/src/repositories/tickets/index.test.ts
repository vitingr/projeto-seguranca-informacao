import { describe, expect, it } from 'vitest'

import {
  parseAdminThreadPage,
  parseAdminTicketDetail,
  parseAuthorList,
  parseQueuePage,
  parseStatusChange,
  parseThreadPage,
  parseTicketDetail,
  parseTicketPage
} from '.'

const at = '2026-11-26T17:04:00.123456Z'

const summary = {
  created_at: at,
  id: 42,
  status: 'aberto',
  title: 'Bug em Portal',
  type: 'bug',
  updated_at: at
}

const page = (items: unknown[]) => ({
  items,
  page: 1,
  page_size: 20,
  pages: 1,
  total: items.length
})

const counts = { aberto: 1, em_andamento: 0, fechado: 0, resolvido: 0 }

const message = {
  created_at: at,
  id: 7,
  kind: 'message',
  mine: true,
  side: 'author',
  text: '<script>alert(1)</script>'
}

const statusLine = {
  created_at: at,
  from: 'aberto',
  id: 9,
  kind: 'status_change',
  mine: false,
  side: 'support',
  to: 'em_andamento'
}

const bugDetail = {
  ...summary,
  data: {
    descricao: 'Erro 500',
    passos: '1. Abra',
    severidade: 'critica',
    sistema: 'Portal'
  }
}

describe('parseTicketPage', () => {
  it('reads a page of the own list', () => {
    expect(parseTicketPage(page([summary]))).toEqual({
      items: [
        {
          createdAt: at,
          id: 42,
          status: 'aberto',
          title: 'Bug em Portal',
          type: 'bug',
          updatedAt: at
        }
      ],
      page: 1,
      pageSize: 20,
      pages: 1,
      total: 1
    })
  })

  it.each([
    ['an extra key', page([{ ...summary, author_login: 'maria' }])],
    ['a missing key', page([{ ...summary, title: undefined }])],
    ['an unknown status', page([{ ...summary, status: 'pendente' }])],
    ['a malformed date', page([{ ...summary, updated_at: 'ontem' }])],
    ['a page key missing', { items: [] }],
    ['not an object', null]
  ])('fails closed on %s', (_name, body) => {
    expect(parseTicketPage(JSON.parse(JSON.stringify(body)))).toBeNull()
  })
})

describe('parseQueuePage and parseAuthorList', () => {
  it('reads the author login and the status counts', () => {
    const queue = parseQueuePage({
      ...page([{ ...summary, author_login: 'maria.souza' }]),
      status_counts: counts
    })
    expect(queue?.items[0]?.authorLogin).toBe('maria.souza')
    expect(queue?.statusCounts).toEqual(counts)
  })

  it('fails closed without the counts or the login', () => {
    expect(parseQueuePage(page([summary]))).toBeNull()
    expect(
      parseQueuePage({ ...page([summary]), status_counts: counts })
    ).toBeNull()
  })

  it('reads the logins', () => {
    expect(parseAuthorList({ logins: ['ana.lima', 'maria'] })).toEqual([
      'ana.lima',
      'maria'
    ])
    expect(parseAuthorList({ logins: ['Ana Lima'] })).toBeNull()
  })
})

describe('parseTicketDetail', () => {
  it('reads the fields of the type', () => {
    expect(parseTicketDetail(bugDetail)?.data).toEqual(bugDetail.data)
  })

  it('accepts null notes on a deploy ticket only', () => {
    const deploy = {
      ...summary,
      data: {
        ambiente: 'prod',
        observacoes: null,
        projeto: 'api',
        repositorio: 'https://github.com/org/api'
      },
      type: 'deploy'
    }
    expect(parseTicketDetail(deploy)?.data.observacoes).toBeNull()
    expect(
      parseTicketDetail({
        ...bugDetail,
        data: { ...bugDetail.data, passos: null }
      })
    ).toBeNull()
  })

  it.each([
    ['data of another type', { ...bugDetail, type: 'acesso' }],
    [
      'an unknown choice',
      { ...bugDetail, data: { ...bugDetail.data, severidade: 'x' } }
    ],
    ['an author in a comum answer', { ...bugDetail, author_login: 'maria' }]
  ])('fails closed on %s', (_name, body) => {
    expect(parseTicketDetail(body)).toBeNull()
  })

  it('the admin detail needs the author login', () => {
    expect(parseAdminTicketDetail(bugDetail)).toBeNull()
    expect(
      parseAdminTicketDetail({ ...bugDetail, author_login: 'maria' })
        ?.authorLogin
    ).toBe('maria')
  })
})

describe('parseThreadPage', () => {
  it('reads messages and status changes, text as typed', () => {
    expect(
      parseThreadPage({ entries: [message, statusLine], has_more: false })
    ).toEqual({
      entries: [
        {
          createdAt: at,
          id: 7,
          kind: 'message',
          mine: true,
          side: 'author',
          text: '<script>alert(1)</script>'
        },
        {
          createdAt: at,
          from: 'aberto',
          id: 9,
          kind: 'status_change',
          mine: false,
          side: 'support',
          to: 'em_andamento'
        }
      ],
      hasMore: false
    })
  })

  it.each([
    ['an actor login in a comum answer', { ...message, actor_login: 'admin' }],
    ['an unknown kind', { ...message, kind: 'note' }],
    ['a message with a status', { ...message, from: 'aberto' }],
    ['an unknown side', { ...message, side: 'admin' }]
  ])('fails closed on %s', (_name, entry) => {
    expect(parseThreadPage({ entries: [entry], has_more: false })).toBeNull()
  })

  it('the admin thread needs the actor login of every entry', () => {
    expect(
      parseAdminThreadPage({ entries: [message], has_more: true })
    ).toBeNull()
    expect(
      parseAdminThreadPage({
        entries: [{ ...message, actor_login: 'maria' }],
        has_more: true
      })?.entries[0]?.actorLogin
    ).toBe('maria')
  })
})

describe('parseStatusChange', () => {
  it('reads the new status', () => {
    expect(
      parseStatusChange({ id: 42, status: 'resolvido', updated_at: at })
    ).toEqual({ id: 42, status: 'resolvido', updatedAt: at })
    expect(
      parseStatusChange({ id: 42, status: 'x', updated_at: at })
    ).toBeNull()
  })
})
