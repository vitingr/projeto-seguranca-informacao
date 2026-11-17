import { describe, expect, it } from 'vitest'

import { cadastroSchema, passwordLength } from './cadastro'

const LOGIN_MESSAGE =
  'Use de 3 a 64 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado.'
const PASSWORD_MESSAGE = 'A senha deve ter de 15 a 128 caracteres.'

const valid = { login: 'maria.silva', papel: 'comum', senha: 'a'.repeat(15) }

const issues = (input: unknown) => {
  const result = cadastroSchema.safeParse(input)
  return result.success
    ? []
    : result.error.issues.map(i => `${i.path.join('.')}: ${i.message}`)
}

describe('cadastroSchema', () => {
  it('accepts valid input and keeps the values as typed', () => {
    expect(cadastroSchema.parse({ ...valid, login: 'Maria.Silva' })).toEqual({
      ...valid,
      login: 'Maria.Silva'
    })
  })

  it.each([['abc'], ['a'.repeat(64)], ['ADMIN'], ['user_1-x.y'], ['123']])(
    'accepts the login %j (after lowercasing)',
    login => {
      expect(issues({ ...valid, login })).toEqual([])
    }
  )

  it.each([
    ['ab'],
    ['a'.repeat(65)],
    [''],
    ['com espaço'],
    [' maria'],
    ['maria@x'],
    ['joão'],
    ['a/b']
  ])('rejects the login %j', login => {
    expect(issues({ ...valid, login })).toEqual([`login: ${LOGIN_MESSAGE}`])
  })

  it.each([
    ['14 characters', 'a'.repeat(14), false],
    ['15 characters', 'a'.repeat(15), true],
    ['128 characters', 'a'.repeat(128), true],
    ['129 characters', 'a'.repeat(129), false],
    // 15 code points, but more UTF-16 units
    ['15 emoji', '😀'.repeat(15), true],
    // NFKC turns the ligature "ﬃ" (1 code point) into "ffi" (3)
    ['13 + ligature = 16 after NFKC', `${'a'.repeat(13)}ﬃ`, true],
    ['12 + ligature = 15 after NFKC', `${'a'.repeat(12)}ﬃ`, true],
    ['11 + ligature = 14 after NFKC', `${'a'.repeat(11)}ﬃ`, false]
  ])('password with %s', (_name, senha, ok) => {
    expect(issues({ ...valid, senha })).toEqual(
      ok ? [] : [`senha: ${PASSWORD_MESSAGE}`]
    )
  })

  it.each([[''], ['root'], ['Admin'], [undefined]])(
    'rejects the role %j',
    papel => {
      expect(issues({ ...valid, papel })).toEqual(['papel: Escolha um papel.'])
    }
  )
})

describe('passwordLength', () => {
  it('counts code points after NFKC', () => {
    expect(passwordLength('😀')).toBe(1)
    expect(passwordLength('ﬃ')).toBe(3)
    expect(passwordLength('é')).toBe(1)
  })
})
