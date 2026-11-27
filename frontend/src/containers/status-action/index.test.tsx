import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { STALE_STATUS } from '@/domain/api-error'

import { StatusAction } from '.'
import { applyStatus, changedText, statusMenuItems } from './rules'

vi.mock('next/navigation', () => ({
  useRouter: () => ({ refresh: () => {} })
}))

const failing = (status: number, code: string) =>
  vi.fn(async () => ({ error: { code, status }, ok: false as const }))

describe('statusMenuItems (spec 003 AC-6.1, BR-3)', () => {
  it('offers the current status (checked) and only the allowed ones', () => {
    expect(statusMenuItems('aberto')).toEqual([
      { current: true, label: 'Aberto', risky: false, status: 'aberto' },
      {
        current: false,
        label: 'Em andamento',
        risky: false,
        status: 'em_andamento'
      },
      { current: false, label: 'Fechado', risky: true, status: 'fechado' }
    ])
    expect(statusMenuItems('resolvido').map(item => item.status)).toEqual([
      'resolvido',
      'em_andamento',
      'fechado'
    ])
  })

  it('has no rows for a closed ticket (AC-6.5)', () => {
    expect(statusMenuItems('fechado')).toEqual([])
  })
})

describe('applyStatus (AC-6.2 to AC-6.6)', () => {
  it('sends the status the admin saw and announces the new one', async () => {
    const change = vi.fn(async () => ({
      data: { id: 7, status: 'em_andamento' as const, updated_at: 'x' },
      ok: true as const
    }))
    const outcome = await applyStatus(
      { change },
      { from: 'aberto', ticketId: 7, to: 'em_andamento' }
    )
    expect(change).toHaveBeenCalledWith({
      de: 'aberto',
      para: 'em_andamento',
      ticketId: 7
    })
    expect(outcome).toEqual({
      kind: 'changed',
      message: 'Status alterado para Em andamento.'
    })
    expect(changedText('fechado')).toBe('Status alterado para Fechado.')
  })

  it('a 409 shows the stale message and reloads the page (AC-6.3)', async () => {
    const outcome = await applyStatus(
      { change: failing(409, STALE_STATUS) },
      { from: 'aberto', ticketId: 7, to: 'em_andamento' }
    )
    expect(outcome).toEqual({
      kind: 'failed',
      message:
        'O status deste ticket foi alterado por outra pessoa. A página foi atualizada; confira antes de tentar de novo.',
      refresh: true
    })
  })

  it('other errors keep the current status on screen (AC-6.6)', async () => {
    const outcome = await applyStatus(
      { change: failing(429, 'too_many_requests') },
      { from: 'aberto', ticketId: 7, to: 'em_andamento' }
    )
    expect(outcome).toEqual({
      kind: 'failed',
      message:
        'Muitas solicitações em pouco tempo. Aguarde um minuto e tente novamente.',
      refresh: false
    })
  })
})

describe('StatusAction', () => {
  it('is a menu trigger "Status: <atual>"', () => {
    const html = renderToStaticMarkup(
      <StatusAction current="em_andamento" ticketId={7} />
    )
    expect(html).toMatch(/aria-haspopup="menu"/)
    expect(html).toContain('Status:')
    expect(html).toContain('>Em andamento</span>')
    expect(html).toContain('rounded-sm')
  })

  it('renders nothing for a closed ticket (AC-6.5)', () => {
    expect(
      renderToStaticMarkup(<StatusAction current="fechado" ticketId={7} />)
    ).toBe('')
  })
})
