import { describe, expect, it } from 'vitest'

import {
  POST_PAGE_MAX,
  bodyHasControl,
  parsePageParam,
  postActionsFor,
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

// 006 D-1, D-2: comum authors edit their own posts, admins only remove, and
// a removed post is frozen for everyone.
describe('postActionsFor', () => {
  const removal = { reason: 'Spam.', removedAt: '2026-11-22T16:00:00Z' }
  it.each([
    ['comum, own post', { mine: true, removal: null }, 'comum', 'edit'],
    ['comum, another post', { mine: false, removal: null }, 'comum', 'none'],
    ['comum, own removed post', { mine: true, removal }, 'comum', 'none'],
    ['admin, a post', { mine: false, removal: null }, 'admin', 'remove'],
    ['admin, a removed post', { mine: false, removal }, 'admin', 'none'],
    [
      'admin, mine (never sent)',
      { mine: true, removal: null },
      'admin',
      'remove'
    ]
  ] as const)('%s → %s', (_, post, role, expected) => {
    expect(postActionsFor(post, role)).toBe(expected)
  })
})
