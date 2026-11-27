import type { Ref } from 'react'

import type { FormFieldControlProps } from '@/components/form-field/types'
import type { ChoiceOption } from '@/domain/tickets/types'

export type ChoiceSelectProps = FormFieldControlProps & {
  labelledBy: string
  name: string
  onBlur: () => void
  onValueChange: (code: string) => void
  options: readonly ChoiceOption[]
  placeholder: string
  ref?: Ref<HTMLButtonElement>
  value: string
}

export interface ChoiceOptionRowProps {
  option: ChoiceOption
}
