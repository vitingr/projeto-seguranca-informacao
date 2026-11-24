// What happens behind the login form, in order.
export const LOGIN_STEPS = [
  {
    tag: 'argon2id',
    text: 'A senha é conferida com o hash Argon2id guardado no banco. Ela nunca fica salva em texto.'
  },
  {
    tag: '__Host-session',
    text: 'O servidor cria a sessão num cookie HttpOnly, fora do alcance do JavaScript da página.'
  },
  {
    tag: '403',
    text: 'Cada página confere o seu papel antes de abrir, e a API confere de novo.'
  },
  {
    tag: 'login_failed',
    text: 'Uma tentativa que falha fica registrada na auditoria, com o IP e o horário.'
  }
] as const

export const STEP_DELAYS = [
  'motion-safe:anim-delay-150',
  'motion-safe:anim-delay-225',
  'motion-safe:anim-delay-300',
  'motion-safe:anim-delay-375'
] as const

export const RISE = 'motion-safe:animate-rise'
