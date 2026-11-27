import type { TicketReviewRow } from '@/components/ticket-review/types'
import {
  choiceLabel,
  fieldsOf,
  ticketDetailRows,
  ticketFieldRows,
  ticketTitle,
  ticketTypeLabel,
  trimField
} from '@/domain/tickets'
import type { TicketInput, TicketType } from '@/domain/tickets/types'

import type { FieldSlot, TicketFormValues } from './types'

const COMPACT = new Set(['choice', 'short', 'url'])

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

export const hasData = (values: TicketFormValues): boolean =>
  Object.values(values).some(value => trimField(value ?? '') !== '')

const filled = (values: TicketFormValues, key: string) =>
  trimField(values[key] ?? '') !== ''

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

const fieldRows = (input: TicketInput): TicketReviewRow[] =>
  ticketFieldRows(input.tipo, { ...input.dados })

export const reviewRows = (input: TicketInput): TicketReviewRow[] =>
  ticketDetailRows(input.tipo, { ...input.dados })

export const progressChips = (input: TicketInput): string[] => [
  ticketTypeLabel(input.tipo),
  ...fieldRows(input)
    .filter(row => row.value !== null)
    .map(row => row.label)
]

export const previewBadges = (
  type: TicketType,
  values: TicketFormValues
): string[] =>
  fieldsOf(type)
    .filter(field => field.kind === 'choice' && filled(values, field.key))
    .map(field => choiceLabel(field, values[field.key]))
