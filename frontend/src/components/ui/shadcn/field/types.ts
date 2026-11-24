import type { ComponentProps } from 'react'

export type FieldProps = ComponentProps<'input'> & {
  error?: string
  /** Shown under the field while there is no error. */
  hint?: string
  id: string
  label: string
}

export interface FormMessageProps {
  children: string
  tone: 'error' | 'success'
}
