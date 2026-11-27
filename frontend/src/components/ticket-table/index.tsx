import Link from 'next/link'

import { StatusBadge } from '@/components/status-badge'
import { formatDateTime } from '@/domain/date-time'
import { LIST_PATHS } from '@/domain/ticket-query'
import type { ListScope } from '@/domain/ticket-query/types'
import { ticketTypeLabel } from '@/domain/tickets'

import type { TicketDateProps, TicketTableProps } from './types'

export const TABLE_TEXTS = {
  author: 'Usuário',
  createdAt: 'Criado em',
  number: '#',
  status: 'Status',
  title: 'Título',
  type: 'Tipo',
  updatedAt: 'Atualizado em'
} as const

export const ticketHref = (scope: ListScope, id: number, listHref: string) => {
  const path = `${LIST_PATHS[scope]}/tickets/${id}`
  return listHref.startsWith(`${LIST_PATHS[scope]}?`)
    ? `${path}?de=${encodeURIComponent(listHref)}`
    : path
}

const TicketDate = ({ iso }: TicketDateProps) => {
  const text = formatDateTime(iso) ?? '—'
  return (
    <time dateTime={iso} title={text}>
      {text}
    </time>
  )
}

const HEAD = 'text-muted-foreground px-3 py-2 text-left text-xs font-medium'
const CELL = 'px-3 py-2.5 align-middle'
const DATE =
  'text-muted-foreground px-3 py-2.5 text-right text-xs whitespace-nowrap'

export const TicketTable = ({
  caption,
  items,
  listHref,
  scope
}: TicketTableProps) => {
  const queue = scope === 'queue'
  return (
    <div className="flex flex-col" data-testid="ticket-table">
      <table className="hidden w-full border-collapse text-sm md:table">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr className="border-border border-b">
            {queue ? (
              <th className={HEAD} scope="col">
                {TABLE_TEXTS.number}
              </th>
            ) : null}
            <th className={HEAD} scope="col">
              {TABLE_TEXTS.title}
            </th>
            <th className={HEAD} scope="col">
              {TABLE_TEXTS.type}
            </th>
            <th className={HEAD} scope="col">
              {TABLE_TEXTS.status}
            </th>
            {queue ? (
              <th className={HEAD} scope="col">
                {TABLE_TEXTS.author}
              </th>
            ) : null}
            <th className={`${HEAD} text-right`} scope="col">
              {TABLE_TEXTS.createdAt}
            </th>
            <th className={`${HEAD} text-right`} scope="col">
              {TABLE_TEXTS.updatedAt}
            </th>
          </tr>
        </thead>
        <tbody className="divide-border divide-y">
          {items.map(ticket => (
            <tr
              className="hover:bg-muted/40 transition-colors"
              data-ticket-id={ticket.id}
              key={ticket.id}
            >
              {queue ? (
                <td className={`${CELL} text-muted-foreground`}>
                  {`#${ticket.id}`}
                </td>
              ) : null}
              <th className={`${CELL} text-left font-normal`} scope="row">
                <span className="flex min-w-0 flex-col gap-0.5">
                  <Link
                    className="focus-visible:ring-ring/50 rounded-sm font-medium break-words outline-none hover:underline focus-visible:ring-3"
                    href={ticketHref(scope, ticket.id, listHref)}
                  >
                    {ticket.title}
                  </Link>
                  {queue ? null : (
                    <span className="text-muted-foreground text-xs">
                      {`#${ticket.id}`}
                    </span>
                  )}
                </span>
              </th>
              <td className={`${CELL} text-muted-foreground`}>
                {ticketTypeLabel(ticket.type)}
              </td>
              <td className={CELL}>
                <StatusBadge status={ticket.status} />
              </td>
              {queue ? (
                <td className={CELL}>
                  <span
                    className="block max-w-40 truncate"
                    title={ticket.authorLogin}
                  >
                    {ticket.authorLogin}
                  </span>
                </td>
              ) : null}
              <td className={DATE}>
                <TicketDate iso={ticket.createdAt} />
              </td>
              <td className={DATE}>
                <TicketDate iso={ticket.updatedAt} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <ul
        aria-label={caption}
        className="divide-border border-border flex flex-col divide-y border-y md:hidden"
      >
        {items.map(ticket => (
          <li
            className="flex flex-col gap-1.5 py-3"
            data-ticket-id={ticket.id}
            key={ticket.id}
          >
            <div className="flex items-baseline justify-between gap-3">
              <Link
                className="focus-visible:ring-ring/50 min-w-0 rounded-sm font-medium break-words outline-none hover:underline focus-visible:ring-3"
                href={ticketHref(scope, ticket.id, listHref)}
              >
                {ticket.title}
              </Link>
              <span className="text-muted-foreground shrink-0 text-xs">
                {`#${ticket.id}`}
              </span>
            </div>
            <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              <StatusBadge status={ticket.status} />
              <span>{ticketTypeLabel(ticket.type)}</span>
              {queue ? (
                <span
                  className="text-foreground max-w-full truncate"
                  title={ticket.authorLogin}
                >
                  {ticket.authorLogin}
                </span>
              ) : null}
              <span>
                {TABLE_TEXTS.updatedAt} <TicketDate iso={ticket.updatedAt} />
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
