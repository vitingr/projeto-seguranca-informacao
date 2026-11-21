import type { z } from 'zod'

import type { cadastroSchema } from './schemas'

export type CadastroInput = z.input<typeof cadastroSchema>
export type CadastroValues = z.output<typeof cadastroSchema>
