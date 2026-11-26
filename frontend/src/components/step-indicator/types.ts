import type { LucideIcon } from 'lucide-react'

export interface StepIndicatorStep {
  /** A short muted line under the label ("Escolha do pedido"). */
  description: string
  icon: LucideIcon
  id: string
  label: string
}

export interface StepIndicatorProps {
  /** Index of the step on screen. */
  current: number
  /** Called with the index of a step already done. */
  onSelect: (index: number) => void
  steps: readonly StepIndicatorStep[]
}
