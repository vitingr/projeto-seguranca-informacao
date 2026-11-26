import type { Step } from './types'

/** The steps in order, with the label of the step indicator. */
export const STEPS = [
  { id: 'tipo', label: 'Tipo' },
  { id: 'detalhes', label: 'Detalhes' },
  { id: 'revisao', label: 'Revisão' }
] as const satisfies readonly { id: Step; label: string }[]

export const BUILDER_MESSAGES = {
  noType: 'Escolha um tipo de ticket.'
} as const
