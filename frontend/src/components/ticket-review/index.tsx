import { cn } from 'cn'

import { DetailList, DetailRow } from '@/components/detail-list'
import { TicketTypeIcon } from '@/components/ticket-type-icon'

import type {
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

const Section = ({ children, hint, title }: ReviewSectionProps) => (
  <section className="flex flex-col gap-3">
    <div className="flex flex-col gap-0.5">
      <h3 className="text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
    <DetailList>{children}</DetailList>
  </section>
)

const value = (row: TicketReviewRow) =>
  row.value === null ? (
    <span className="text-muted-foreground">{REVIEW_TEXTS.notInformed}</span>
  ) : (
    <span className={cn(row.risky && 'text-destructive')}>{row.value}</span>
  )

export const TicketReview = ({
  rows,
  title,
  type,
  typeLabel
}: TicketReviewProps) => (
  <div className="flex flex-col gap-6">
    <Section hint={REVIEW_TEXTS.requestHint} title={REVIEW_TEXTS.request}>
      <DetailRow label={REVIEW_TEXTS.type}>
        <span className="flex items-center gap-2">
          <TicketTypeIcon className="size-7" type={type} />
          {typeLabel}
        </span>
      </DetailRow>
      <DetailRow label={REVIEW_TEXTS.title} preserveLines>
        {title}
      </DetailRow>
    </Section>
    <Section hint={REVIEW_TEXTS.detailsHint} title={REVIEW_TEXTS.details}>
      {rows.map(row => (
        <DetailRow key={row.key} label={row.label} preserveLines>
          {value(row)}
        </DetailRow>
      ))}
    </Section>
  </div>
)
