import type { TicketReviewRow } from '@/components/ticket-review/types'
import {
  choiceLabel,
  fieldsOf,
  ticketTitle,
  ticketTypeLabel,
  trimField
} from '@/domain/tickets'
import type { TicketInput, TicketType } from '@/domain/tickets/types'

import type { FieldSlot, TicketFormValues } from './types'

const COMPACT = new Set(['choice', 'short', 'url'])

/**
 * The fields of a type in screen order (plan 5.4): short fields, URLs and
 * choices first, side by side from `md`, then the long ones across the row.
 * When the compact fields are odd, the last one takes the whole row. The
 * order on screen is the DOM order, so the tab order follows it.
 */
export const fieldLayout = (type: TicketType): FieldSlot[] => {
  const fields = fieldsOf(type)
  const compact = fields.filter(field => COMPACT.has(field.kind))
  const wide = fields.filter(field => !COMPACT.has(field.kind))
  return [
    ...compact.map((field, index) => ({
      field,
      wide: compact.length % 2 === 1 && index === compact.length - 1
    })),
    ...wide.map(field => ({ field, wide: true }))
  ]
}

/** Whether the form holds anything typed or chosen (AC-2.4, AC-3.8). */
export const hasData = (values: TicketFormValues): boolean =>
  Object.values(values).some(value => trimField(value ?? '') !== '')

const filled = (values: TicketFormValues, key: string) =>
  trimField(values[key] ?? '') !== ''

/**
 * The title the ticket will get (BR-3), for the preview; `null` while the
 * fields it is built from are empty.
 */
export const previewTitle = (
  type: TicketType,
  values: TicketFormValues
): null | string => {
  const needs = {
    acesso: ['sistema'],
    bug: ['sistema'],
    deploy: ['projeto', 'ambiente'],
    outro: ['titulo']
  }[type]
  if (!needs.every(key => filled(values, key))) return null
  return ticketTitle({ dados: values, tipo: type } as unknown as TicketInput)
}

/**
 * Every field of the type in screen order, with the value that will be sent
 * (trimmed by the schema), the label of a fixed choice, and `null` for an
 * optional field left empty.
 */
const fieldRows = (input: TicketInput): TicketReviewRow[] => {
  const dados: Partial<Record<string, string>> = { ...input.dados }
  return fieldLayout(input.tipo).map(({ field }) => {
    const raw = dados[field.key]
    const option = field.options?.find(item => item.code === raw)
    return {
      key: field.key,
      label: field.label,
      risky: option?.risky ?? false,
      value: raw === undefined || raw === '' ? null : (option?.label ?? raw)
    }
  })
}

/** The field whose value is the ticket title itself ("Outro", BR-3). */
const TITLE_FIELDS: Partial<Record<TicketType, string>> = { outro: 'titulo' }

/**
 * The rows of the "Detalhes" group of the review (AC-4.1). The field that is
 * the ticket title ("Título" of "Outro") is left out: the "Pedido" group
 * already shows it as the title, so it is shown once.
 */
export const reviewRows = (input: TicketInput): TicketReviewRow[] =>
  fieldRows(input).filter(row => row.key !== TITLE_FIELDS[input.tipo])

/**
 * The chips of the send progress: the type, then the labels of every field
 * sent with a value, the title field included. Names only, never what was
 * typed (AC-5.5).
 */
export const progressChips = (input: TicketInput): string[] => [
  ticketTypeLabel(input.tipo),
  ...fieldRows(input)
    .filter(row => row.value !== null)
    .map(row => row.label)
]

/** The labels of the options chosen so far, for the preview badges. */
export const previewBadges = (
  type: TicketType,
  values: TicketFormValues
): string[] =>
  fieldsOf(type)
    .filter(field => field.kind === 'choice' && filled(values, field.key))
    .map(field => choiceLabel(field, values[field.key]))
