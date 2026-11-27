import { z } from 'zod'

import { LOGIN_PATTERN, passwordLength } from '@/domain/credentials'
import { ROLES } from '@/domain/roles'

export const cadastroSchema = z.object({
  login: z
    .string()
    .refine(
      value => LOGIN_PATTERN.test(value.toLowerCase()),
      'Use de 3 a 64 caracteres: letras minúsculas, números, ponto, hífen ou sublinhado.'
    ),
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
