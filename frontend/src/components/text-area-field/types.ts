import type { Ref, TextareaHTMLAttributes } from 'react'

export type TextAreaFieldProps = Omit<
  TextareaHTMLAttributes<HTMLTextAreaElement>,
  'id'
> & {
  count: number
  hint?: string
  id: string
  max: number
  ref?: Ref<HTMLTextAreaElement>
}
