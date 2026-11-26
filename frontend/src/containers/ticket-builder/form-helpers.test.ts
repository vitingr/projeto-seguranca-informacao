import { describe, expect, it } from 'vitest'

import {
  fieldLayout,
  hasData,
  previewBadges,
  previewTitle,
  progressChips,
  reviewRows
} from './form-helpers'

const layout = (type: Parameters<typeof fieldLayout>[0]) =>
  fieldLayout(type).map(({ field, wide }) => `${field.key}${wide ? '*' : ''}`)

describe('fieldLayout', () => {
  it('puts compact fields first, side by side, and long ones across', () => {
    expect(layout('acesso')).toEqual([
      'sistema',
      'nivel_acesso',
      'justificativa*'
    ])
    expect(layout('bug')).toEqual([
      'sistema',
      'severidade',
      'descricao*',
      'passos*'
    ])
    expect(layout('deploy')).toEqual([
      'projeto',
      'repositorio',
      'ambiente*',
      'observacoes*'
    ])
    expect(layout('outro')).toEqual(['titulo*', 'descricao*'])
  })
})

describe('hasData', () => {
  it('is false for empty and whitespace-only values', () => {
    expect(hasData({})).toBe(false)
    expect(hasData({ descricao: '', titulo: '  \n ' })).toBe(false)
  })

  it('is true for any typed or chosen value', () => {
    expect(hasData({ descricao: '', titulo: 'x' })).toBe(true)
    expect(hasData({ severidade: 'alta' })).toBe(true)
  })
})

describe('previewTitle', () => {
  it('is null while its fields are empty', () => {
    expect(previewTitle('acesso', { sistema: ' ' })).toBeNull()
    expect(previewTitle('deploy', { ambiente: '', projeto: 'api' })).toBeNull()
    expect(previewTitle('outro', { titulo: '' })).toBeNull()
  })

  it('follows the title rule of the server (BR-3)', () => {
    expect(previewTitle('acesso', { sistema: ' Grafana ' })).toBe(
      'Acesso a Grafana'
    )
    expect(previewTitle('bug', { sistema: 'Portal' })).toBe('Bug em Portal')
    expect(
      previewTitle('deploy', { ambiente: 'prod', projeto: 'api-pagamentos' })
    ).toBe('Deploy de api-pagamentos em Produção')
    expect(previewTitle('outro', { titulo: 'Liberar porta' })).toBe(
      'Liberar porta'
    )
  })
})

describe('reviewRows', () => {
  it('lists every field in screen order with labels for choices', () => {
    expect(
      reviewRows({
        dados: {
          descricao: 'Erro 500',
          passos: '1. Abra\n2. Salve',
          severidade: 'critica',
          sistema: 'Portal'
        },
        tipo: 'bug'
      })
    ).toEqual([
      {
        key: 'sistema',
        label: 'Sistema afetado',
        risky: false,
        value: 'Portal'
      },
      { key: 'severidade', label: 'Severidade', risky: true, value: 'Crítica' },
      { key: 'descricao', label: 'Descrição', risky: false, value: 'Erro 500' },
      {
        key: 'passos',
        label: 'Passos para reproduzir',
        risky: false,
        value: '1. Abra\n2. Salve'
      }
    ])
  })

  it('gives null for the optional notes left out (AC-4.1)', () => {
    const rows = reviewRows({
      dados: {
        ambiente: 'homolog',
        projeto: 'api',
        repositorio: 'https://github.com/org/api'
      },
      tipo: 'deploy'
    })
    expect(rows.find(row => row.key === 'observacoes')?.value).toBeNull()
    expect(rows.find(row => row.key === 'ambiente')).toMatchObject({
      risky: false,
      value: 'Homologação'
    })
  })
})

describe('progressChips', () => {
  it('names the type and the fields with a value, never the values (AC-5.5)', () => {
    const chips = progressChips({
      dados: {
        ambiente: 'prod',
        projeto: 'api-secreta',
        repositorio: 'https://github.com/org/api'
      },
      tipo: 'deploy'
    })
    expect(chips).toEqual([
      'Deploy de projeto',
      'Nome do projeto',
      'Repositório',
      'Ambiente'
    ])
    expect(chips.join(' ')).not.toMatch(/api-secreta|github|Produção/)
  })
})

describe('previewBadges', () => {
  it('lists the labels of the chosen options only', () => {
    expect(previewBadges('bug', { severidade: '', sistema: 'x' })).toEqual([])
    expect(previewBadges('bug', { severidade: 'critica' })).toEqual(['Crítica'])
    expect(previewBadges('deploy', { ambiente: 'homolog' })).toEqual([
      'Homologação'
    ])
    expect(previewBadges('acesso', { nivel_acesso: 'leitura' })).toEqual([
      'Leitura'
    ])
  })
})
