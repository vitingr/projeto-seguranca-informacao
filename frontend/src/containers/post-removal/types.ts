import type { z } from 'zod'

import type { removalSchema } from './schemas'

export type RemovalValues = z.output<typeof removalSchema>

export interface PostRemovalProps {
  /** Closes the form without removing. */
  onCancel: () => void
  postId: number
}
