import { describe, expect, it } from 'vitest'

import {
  fieldLayout,
  hasData,
  previewBadges,
  previewTitle
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
