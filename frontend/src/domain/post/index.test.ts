import { describe, expect, it } from 'vitest'

import {
  POST_PAGE_MAX,
  bodyHasControl,
  parsePageParam,
  postLength,
  titleHasControl
} from '.'

describe('postLength', () => {
  it.each([
    ['', 0],
    ['   ', 0],
    ['  abc  ', 3],
    ['é'.repeat(120), 120],
    ['😀'.repeat(3), 3],
    ['\nlinha\n', 5]
  ])('%j has %i code points', (value, expected) => {
    expect(postLength(value)).toBe(expected)
  })
})

describe('control characters', () => {
  it('titles allow none', () => {
    expect(titleHasControl('Título normal')).toBe(false)
    for (const c of [
      '\n',
      '\r',
      '\t',
      '\u0000',
      '\u001b',
      '\u007f',
      '\u0085'
    ]) {
      expect(titleHasControl(`a${c}b`)).toBe(true)
    }
  })

  it('bodies allow only line breaks and tab', () => {
    expect(bodyHasControl('linha 1\r\nlinha 2\n\tfim')).toBe(false)
    for (const c of ['\u0000', '\u001b', '\u007f', '\u0085']) {
      expect(bodyHasControl(`a${c}b`)).toBe(true)
    }
  })
})

describe('parsePageParam', () => {
  it.each<[string | string[] | undefined, number]>([
    [undefined, 1],
    ['1', 1],
    ['2', 2],
    [String(POST_PAGE_MAX), POST_PAGE_MAX],
    [String(POST_PAGE_MAX + 1), 1],
    ['0', 1],
    ['-1', 1],
    ['01', 1],
    ['abc', 1],
    ['1.5', 1],
    [['2', '3'], 1]
  ])('%j is page %i', (raw, expected) => {
    expect(parsePageParam(raw)).toBe(expected)
  })
})
