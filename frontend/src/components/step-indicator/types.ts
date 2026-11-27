import type { LucideIcon } from 'lucide-react'

export interface StepIndicatorStep {
  description: string
  icon: LucideIcon
  id: string
  label: string
}

export interface StepIndicatorProps {
  current: number
  locked?: boolean
  onSelect: (index: number) => void
  steps: readonly StepIndicatorStep[]
}
