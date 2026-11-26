import type { Ref, TextareaHTMLAttributes } from 'react'

export type TextAreaFieldProps = Omit<
  TextareaHTMLAttributes<HTMLTextAreaElement>,
  'id'
> & {
  /** Characters typed, in code points (spec section 4). */
  count: number
  /** Help text in the strip under the box. */
  hint?: string
  id: string
  max: number
  ref?: Ref<HTMLTextAreaElement>
}
