'use client'

import { useRouter } from 'next/navigation'
import { useId, useTransition } from 'react'

import { ChoiceSelect } from '@/components/choice-select'
import { LOGIN_PATTERN } from '@/domain/credentials'
import { listQueryHref } from '@/domain/ticket-query'
import type { ListQuery } from '@/domain/ticket-query/types'
import { TICKET_TYPES } from '@/domain/tickets'
import type { ChoiceOption, TicketType } from '@/domain/tickets/types'

import type { SelectFilter, TicketListFiltersProps } from './types'

export const FILTER_TEXTS = {
  all: 'Todos',
  busy: 'Atualizando a lista…',
  type: 'Tipo',
  user: 'Usuário'
} as const

/** The code of "Todos": not a type code and never a valid login. */
export const ALL = '*'

const TYPE_CODES: readonly string[] = TICKET_TYPES.map(type => type.code)

const TYPE_OPTIONS: readonly ChoiceOption[] = [
  { code: ALL, description: '', label: FILTER_TEXTS.all },
  ...TICKET_TYPES.map(type => ({
    code: type.code,
    description: '',
    label: type.label
  }))
]

/**
 * The address after choosing `code` in a Select filter, back on page 1
 * (AC-1.5), or `null` when the code is not one the filter offers, so an
 * unknown value never reaches the address (AC-1.6).
 */
export const filterHref = (
  query: ListQuery,
  filter: SelectFilter,
  code: string,
  logins: readonly string[] = []
): null | string => {
  if (filter === 'type') {
    if (code === ALL) return listQueryHref(query, { type: null })
    return TYPE_CODES.includes(code)
      ? listQueryHref(query, { type: code as TicketType })
      : null
  }
  if (query.scope !== 'queue') return null
  if (code === ALL) return listQueryHref(query, { user: null })
  return logins.includes(code) && LOGIN_PATTERN.test(code)
    ? listQueryHref(query, { user: code })
    : null
}

/**
 * The filter row of a list (plan 6.3, P-13): the status links, then the
 * type Select (and the user Select in the queue) over `choice-select`. A
 * choice navigates with `router.push` inside a transition; while the new
 * page loads, the list under the row is marked busy and the filters stay
 * usable.
 */
export const TicketListFilters = ({
  children,
  logins,
  query,
  statusFilter
}: TicketListFiltersProps) => {
  const router = useRouter()
  const [pending, startTransition] = useTransition()
  const typeId = useId()
  const userId = useId()

  const choose = (filter: SelectFilter, code: string) => {
    const href = filterHref(query, filter, code, logins)
    if (href === null) return
    startTransition(() => {
      router.push(href)
    })
  }

  const userOptions: ChoiceOption[] = [
    { code: ALL, description: '', label: FILTER_TEXTS.all },
    ...(logins ?? []).map(login => ({
      code: login,
      description: '',
      label: login
    }))
  ]

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        {statusFilter}
        <div className="grid gap-3 sm:grid-cols-2 lg:flex lg:items-end">
          <div className="flex flex-col gap-1.5 lg:w-56">
            <span
              className="text-muted-foreground text-xs font-medium"
              id={`${typeId}-label`}
            >
              {FILTER_TEXTS.type}
            </span>
            <ChoiceSelect
              id={typeId}
              labelledBy={`${typeId}-label`}
              name="tipo"
              onBlur={() => {}}
              onValueChange={code => choose('type', code)}
              options={TYPE_OPTIONS}
              placeholder={FILTER_TEXTS.all}
              value={query.type ?? ALL}
            />
          </div>
          {logins ? (
            <div className="flex flex-col gap-1.5 lg:w-56">
              <span
                className="text-muted-foreground text-xs font-medium"
                id={`${userId}-label`}
              >
                {FILTER_TEXTS.user}
              </span>
              <ChoiceSelect
                id={userId}
                labelledBy={`${userId}-label`}
                name="usuario"
                onBlur={() => {}}
                onValueChange={code => choose('user', code)}
                options={userOptions}
                placeholder={FILTER_TEXTS.all}
                value={query.user ?? ALL}
              />
            </div>
          ) : null}
        </div>
      </div>
      <p aria-live="polite" className="sr-only">
        {pending ? FILTER_TEXTS.busy : ''}
      </p>
      <div
        aria-busy={pending || undefined}
        className="flex flex-col gap-4 data-busy:cursor-progress data-busy:opacity-60 motion-safe:transition-opacity"
        data-busy={pending || undefined}
        data-testid="ticket-list-body"
      >
        {children}
      </div>
    </div>
  )
}
