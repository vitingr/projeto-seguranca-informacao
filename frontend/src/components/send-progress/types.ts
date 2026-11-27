export type SendPhase = 'done' | 'failed' | 'registering'

export type SendStepState = 'active' | 'done' | 'failed'

export interface SendProgressProps {
  chips: readonly string[]
  elapsedMs: number
  enterFrom?: number
  phase: SendPhase
}
