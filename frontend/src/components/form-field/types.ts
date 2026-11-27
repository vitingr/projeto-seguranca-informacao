import type { InputHTMLAttributes, ReactNode, Ref } from 'react'

export interface FormFieldControlProps {
  'aria-describedby'?: string
  'aria-invalid'?: true
  'aria-required'?: true
  id: string
}

export interface FormFieldProps {
  children: (control: FormFieldControlProps) => ReactNode
  className?: string
  error?: string
  hint?: string
  id: string
  label: string
  labelId?: string
  required: boolean
}

export type FormInputProps = InputHTMLAttributes<HTMLInputElement> & {
  ref?: Ref<HTMLInputElement>
}
