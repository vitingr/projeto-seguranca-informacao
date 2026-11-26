import type { Ref } from 'react'

import type { FormFieldControlProps } from '@/components/form-field/types'
import type { ChoiceOption } from '@/domain/tickets/types'

export type ChoiceSelectProps = FormFieldControlProps & {
  /** The id of the visible label that names the trigger. */
  labelledBy: string
  name: string
  onBlur: () => void
  /** Called with the chosen code. */
  onValueChange: (code: string) => void
  options: readonly ChoiceOption[]
  placeholder: string
  ref?: Ref<HTMLButtonElement>
  /** The chosen code, or `''` when none is chosen. */
  value: string
}

export interface ChoiceOptionRowProps {
  option: ChoiceOption
}
