/**
 * What really happened: the request is pending, the server created the
 * ticket, or it answered with an error (or never answered).
 */
export type SendPhase = 'done' | 'failed' | 'registering'

export type SendStepState = 'active' | 'done' | 'failed'

export interface SendProgressProps {
  /** Names shown as chips: the type, then the labels of the fields sent. */
  chips: readonly string[]
  /** Milliseconds since the click, measured by the page. */
  elapsedMs: number
  /**
   * Rows from this index on enter with the blur fade; earlier ones are
   * drawn in place (on the confirmation, only the rows new there move).
   */
  enterFrom?: number
  phase: SendPhase
}
