'use client'

import { ChevronDownIcon, ChevronUpIcon } from 'lucide-react'
import { useId, useState } from 'react'

import { Button } from '@/components/ui/shadcn/button'

import type { SideCardProps } from './types'

/**
 * A small bordered card of the side column ("Dicas", "Prévia"), as in
 * `design/references/tela-criacao-tickets.png`: an icon, a title and a
 * button on a light gray square that collapses the body.
 */
export const SideCard = ({
  children,
  defaultOpen = true,
  icon: Icon,
  title
}: SideCardProps) => {
  const [open, setOpen] = useState(defaultOpen)
  const id = useId()
  const bodyId = `${id}-body`
  const titleId = `${id}-title`

  return (
    <section
      aria-labelledby={titleId}
      className="border-border flex flex-col gap-3 rounded-sm border p-4"
    >
      <div className="flex items-center justify-between gap-2">
        <h2
          className="flex items-center gap-2 text-sm font-medium"
          id={titleId}
        >
          <Icon className="text-muted-foreground size-4" aria-hidden />
          {title}
        </h2>
        <Button
          aria-controls={bodyId}
          aria-expanded={open}
          aria-label={open ? `Recolher ${title}` : `Mostrar ${title}`}
          className="bg-muted"
          onClick={() => setOpen(value => !value)}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          {open ? (
            <ChevronUpIcon aria-hidden />
          ) : (
            <ChevronDownIcon aria-hidden />
          )}
        </Button>
      </div>
      <div hidden={!open} id={bodyId}>
        {children}
      </div>
    </section>
  )
}
