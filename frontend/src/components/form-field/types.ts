import type { InputHTMLAttributes, ReactNode, Ref } from 'react'

/** What the control needs to be tied to its label, hint and error. */
export interface FormFieldControlProps {
  'aria-describedby'?: string
  'aria-invalid'?: true
  'aria-required'?: true
  id: string
}

export interface FormFieldProps {
  /** Renders the control with the ids and states it needs. */
  children: (control: FormFieldControlProps) => ReactNode
  className?: string
  error?: string
  /** Help text under the label (only rules, plan 5.4). */
  hint?: string
  id: string
  label: string
  /** The id of the label, for controls named with `aria-labelledby`. */
  labelId?: string
  required: boolean
}

export type FormInputProps = InputHTMLAttributes<HTMLInputElement> & {
  ref?: Ref<HTMLInputElement>
}
