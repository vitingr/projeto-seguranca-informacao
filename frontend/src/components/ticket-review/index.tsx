import { cn } from 'cn'

import { TicketTypeIcon } from '@/components/ticket-type-icon'

import type {
  ReviewRowProps,
  ReviewSectionProps,
  TicketReviewProps,
  TicketReviewRow
} from './types'

export const REVIEW_TEXTS = {
  details: 'Detalhes',
  detailsHint: 'Os campos exatamente como serão enviados.',
  notInformed: 'Não informado',
  request: 'Pedido',
  requestHint: 'O tipo e o título que a equipe vai ver na fila.',
  title: 'Título',
  type: 'Tipo'
} as const

const Row = ({ children, label }: ReviewRowProps) => (
  <div className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:gap-6">
    <dt className="text-muted-foreground text-sm sm:w-44 sm:shrink-0">
      {label}
    </dt>
    <dd className="min-w-0 flex-1 text-sm break-words whitespace-pre-wrap">
      {children}
    </dd>
  </div>
)

const Section = ({ children, hint, title }: ReviewSectionProps) => (
  <section className="flex flex-col gap-3">
    <div className="flex flex-col gap-0.5">
      <h3 className="text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
    <dl className="border-border divide-border flex flex-col divide-y rounded-sm border">
      {children}
    </dl>
  </section>
)

const value = (row: TicketReviewRow) =>
  row.value === null ? (
    <span className="text-muted-foreground">{REVIEW_TEXTS.notInformed}</span>
  ) : (
    <span className={cn(row.risky && 'text-destructive')}>{row.value}</span>
  )

/**
 * The review step (spec US-4), in the grouped rows of
 * `design/references/settings.png`: a "Pedido" group with the type and the
 * title the ticket will get, and a "Detalhes" group with every field. Values
 * are plain text with their line breaks kept; nothing is read as HTML or
 * markdown (AC-4.1, AC-4.3, S-8).
 */
export const TicketReview = ({
  rows,
  title,
  type,
  typeLabel
}: TicketReviewProps) => (
  <div className="flex flex-col gap-6">
    <Section hint={REVIEW_TEXTS.requestHint} title={REVIEW_TEXTS.request}>
      <Row label={REVIEW_TEXTS.type}>
        <span className="flex items-center gap-2">
          <TicketTypeIcon className="size-7" type={type} />
          {typeLabel}
        </span>
      </Row>
      <Row label={REVIEW_TEXTS.title}>{title}</Row>
    </Section>
    <Section hint={REVIEW_TEXTS.detailsHint} title={REVIEW_TEXTS.details}>
      {rows.map(row => (
        <Row key={row.key} label={row.label}>
          {value(row)}
        </Row>
      ))}
    </Section>
  </div>
)
