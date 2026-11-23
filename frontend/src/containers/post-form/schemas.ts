import { z } from 'zod'

import {
  POST_BODY_MAX,
  POST_TITLE_MAX,
  bodyHasControl,
  postLength,
  titleHasControl
} from '@/domain/post'

// The same rules as the backend (005 D-4), checked before sending so the
// user gets a field message. The backend stays the authority: it answers
// 422 for anything these rules miss. Values are sent as typed; the backend
// trims them itself.

export const POST_MESSAGES = {
  bodyControl: 'O texto tem caracteres de controle não permitidos.',
  bodyEmpty: 'Escreva o texto do post.',
  bodyTooLong: `O texto deve ter no máximo ${POST_BODY_MAX} caracteres.`,
  titleControl: 'O título não pode ter caracteres de controle.',
  titleEmpty: 'Informe um título.',
  titleTooLong: `O título deve ter no máximo ${POST_TITLE_MAX} caracteres.`
} as const

export const postSchema = z.object({
  body: z
    .string()
    .refine(value => postLength(value) >= 1, POST_MESSAGES.bodyEmpty)
    .refine(
      value => postLength(value) <= POST_BODY_MAX,
      POST_MESSAGES.bodyTooLong
    )
    .refine(value => !bodyHasControl(value), POST_MESSAGES.bodyControl),
  title: z
    .string()
    .refine(value => postLength(value) >= 1, POST_MESSAGES.titleEmpty)
    .refine(
      value => postLength(value) <= POST_TITLE_MAX,
      POST_MESSAGES.titleTooLong
    )
    .refine(value => !titleHasControl(value), POST_MESSAGES.titleControl)
})
