import { describe, expect, it } from 'vitest'

import { loginSchema } from './login'

const messages = (input: unknown) => {
  const result = loginSchema.safeParse(input)
  return result.success
    ? []
    : result.error.issues.map(i => `${i.path.join('.')}: ${i.message}`)
}

describe('loginSchema', () => {
  it('accepts filled fields and trims only the login', () => {
    expect(loginSchema.parse({ login: '  Admin ', senha: ' p a ss ' })).toEqual(
      {
        login: 'Admin',
        senha: ' p a ss '
      }
    )
  })

  it.each([
    [{ login: '', senha: 'x' }, ['login: Informe o login.']],
    [{ login: '   ', senha: 'x' }, ['login: Informe o login.']],
    [{ login: 'a', senha: '' }, ['senha: Informe a senha.']],
    [
      { login: '', senha: '' },
      ['login: Informe o login.', 'senha: Informe a senha.']
    ]
  ])('rejects %j', (input, want) => {
    expect(messages(input)).toEqual(want)
  })

  it('does not apply the registration password rules', () => {
    expect(messages({ login: 'a', senha: 'short' })).toEqual([])
  })
})
