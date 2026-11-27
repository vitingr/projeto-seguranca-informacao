'use client'

import { useEffect, useState } from 'react'

import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger
} from '@/components/ui/shadcn/tabs'
import { TICKET_TABS, readTicketTab } from '@/domain/ticket-query'
import type { TicketTab } from '@/domain/ticket-query/types'

import type { TicketTabsProps } from './types'

export const TAB_TEXTS = {
  conversa: 'Conversa',
  detalhes: 'Detalhes',
  label: 'Seções do ticket'
} as const

/**
 * The address of the page with `tab` selected: `?aba=conversa`, and no
 * `aba` for "Detalhes"; every other parameter (`de`) is kept as it is.
 */
export const tabHref = (href: string, tab: TicketTab): string => {
  const url = new URL(href)
  if (tab === 'detalhes') url.searchParams.delete('aba')
  else url.searchParams.set('aba', tab)
  return `${url.pathname}${url.search}${url.hash}`
}

/**
 * The two tabs of a ticket page (spec 003 AC-2.1, AC-2.3, plan 6.4): the
 * WAI-ARIA tabs of Base UI (arrow keys between tabs, each panel labelled by
 * its tab), with both panels rendered by the server and kept mounted. The
 * chosen tab goes to the address with `history.replaceState`, only when the
 * address really changes (001 T9.9), so no request is made. The title gets
 * the focus when the page opens.
 */
export const TicketTabs = ({
  conversation,
  details,
  initialTab,
  titleId
}: TicketTabsProps) => {
  const [tab, setTab] = useState<TicketTab>(initialTab)

  useEffect(() => {
    document.getElementById(titleId)?.focus({ preventScroll: true })
  }, [titleId])

  const choose = (value: unknown) => {
    const next = readTicketTab(value)
    setTab(next)
    const current = window.location.href
    const href = tabHref(current, next)
    const { hash, pathname, search } = window.location
    if (href !== `${pathname}${search}${hash}`) {
      window.history.replaceState(window.history.state, '', href)
    }
  }

  return (
    <Tabs onValueChange={choose} value={tab}>
      <TabsList aria-label={TAB_TEXTS.label}>
        {TICKET_TABS.map(value => (
          <TabsTrigger className="px-4" key={value} value={value}>
            {TAB_TEXTS[value]}
          </TabsTrigger>
        ))}
      </TabsList>
      <TabsContent className="pt-4" value="detalhes" keepMounted>
        {details}
      </TabsContent>
      <TabsContent className="pt-4" value="conversa" keepMounted>
        {conversation}
      </TabsContent>
    </Tabs>
  )
}
