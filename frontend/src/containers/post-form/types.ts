import type { ReactNode } from 'react'
import type { z } from 'zod'

import type { Post } from '@/domain/post/types'

import type { postSchema } from './schemas'

export type PostValues = z.output<typeof postSchema>

export type PostFormProps =
  | { mode: 'create' }
  | { mode: 'edit'; onCancel: () => void; onSaved: () => void; post: Post }

export interface FieldErrorProps {
  children: ReactNode
  id: string
}
