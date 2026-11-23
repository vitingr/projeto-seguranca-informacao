import { z } from 'zod'

import { REMOVAL_REASON_MAX, postLength, reasonHasControl } from '@/domain/post'

// The same rules as the backend (006 D-2): 1 to 500 code points after
// trimming, and no control characters other than line breaks and tab. The
// backend stays the authority and answers 422 for anything these miss.

export const REMOVAL_MESSAGES = {
  reasonControl: 'O motivo tem caracteres de controle não permitidos.',
  reasonEmpty: 'Informe o motivo da remoção.',
  reasonTooLong: `O motivo deve ter no máximo ${REMOVAL_REASON_MAX} caracteres.`
} as const

export const removalSchema = z.object({
  reason: z
    .string()
    .refine(value => postLength(value) >= 1, REMOVAL_MESSAGES.reasonEmpty)
    .refine(
      value => postLength(value) <= REMOVAL_REASON_MAX,
      REMOVAL_MESSAGES.reasonTooLong
    )
    .refine(value => !reasonHasControl(value), REMOVAL_MESSAGES.reasonControl)
})
