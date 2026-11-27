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

export const tabHref = (href: string, tab: TicketTab): string => {
  const url = new URL(href)
  if (tab === 'detalhes') url.searchParams.delete('aba')
  else url.searchParams.set('aba', tab)
  return `${url.pathname}${url.search}${url.hash}`
}

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
