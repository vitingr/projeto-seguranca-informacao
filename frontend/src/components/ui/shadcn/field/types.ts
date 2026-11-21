import type { ComponentProps } from 'react'

export type FieldProps = ComponentProps<'input'> & {
  error?: string
  id: string
  label: string
}

export interface FormMessageProps {
  children: string
  tone: 'error' | 'success'
}
