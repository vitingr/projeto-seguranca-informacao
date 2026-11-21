import type { z } from 'zod'

import type { loginSchema } from './schemas'

export type LoginValues = z.infer<typeof loginSchema>

export interface LoginFormProps {
  expired: boolean
  next?: string
}
