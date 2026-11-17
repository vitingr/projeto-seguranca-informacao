import { z } from 'zod'

import { ROLES } from '@/shared/api/types'

// The same rules as the backend (002 D-14, 003 D-7), checked before sending
// so the user gets a field message. The backend stays the authority: it
// answers 422 for anything these rules miss. Values are sent as typed; the
// backend lowercases the login itself.

export const LOGIN_PATTERN = /^[a-z0-9._-]{3,64}$/

/** Password length as the backend counts it: code points after NFKC. */
export const passwordLength = (value: string) =>
  [...value.normalize('NFKC')].length

export const cadastroSchema = z.object({
  login: z
    .string()
    .refine(
      value => LOGIN_PATTERN.test(value.toLowerCase()),
      'Use de 3 a 64 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado.'
    ),
  // A string in (the empty "Escolha…" option is ''), a Role out.
  papel: z
    .string({ error: 'Escolha um papel.' })
    .pipe(z.enum(ROLES, { error: 'Escolha um papel.' })),
  senha: z
    .string()
    .refine(
      value => passwordLength(value) >= 15 && passwordLength(value) <= 128,
      'A senha deve ter de 15 a 128 caracteres.'
    )
})

export type CadastroInput = z.input<typeof cadastroSchema>
export type CadastroValues = z.output<typeof cadastroSchema>
