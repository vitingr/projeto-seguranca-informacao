import { cn } from 'cn'
import { ChevronLeftIcon, ChevronRightIcon } from 'lucide-react'
import Link from 'next/link'

import { buttonVariants } from '@/components/ui/shadcn/button'
import { listQueryHref } from '@/domain/ticket-query'

import type { PagerProps } from './types'

export const PAGER_TEXTS = {
  first: 'Ir para a primeira página',
  label: 'Páginas',
  missing: 'Esta página não existe.',
  next: 'Próxima',
  previous: 'Anterior'
} as const

const pageOf = (page: number, pages: number) => `Página ${page} de ${pages}`

const END =
  'text-muted-foreground inline-flex h-8 items-center gap-1 px-2.5 text-sm'

export const Pager = ({ pages, query }: PagerProps) => {
  const { page } = query
  if (page > pages) {
    return (
      <div className="flex flex-col items-start gap-3">
        <p className="text-muted-foreground text-sm">{PAGER_TEXTS.missing}</p>
        <Link
          className={cn(buttonVariants({ variant: 'outline' }))}
          href={listQueryHref(query, { page: 1 })}
        >
          {PAGER_TEXTS.first}
        </Link>
      </div>
    )
  }
  return (
    <nav aria-label={PAGER_TEXTS.label}>
      <ul className="flex items-center justify-between gap-2 sm:justify-end">
        <li>
          {page > 1 ? (
            <Link
              className={cn(buttonVariants({ variant: 'ghost' }), 'px-2.5')}
              href={listQueryHref(query, { page: page - 1 })}
              rel="prev"
            >
              <ChevronLeftIcon aria-hidden />
              {PAGER_TEXTS.previous}
            </Link>
          ) : (
            <span className={END}>
              <ChevronLeftIcon className="size-4" aria-hidden />
              {PAGER_TEXTS.previous}
            </span>
          )}
        </li>
        <li className="text-muted-foreground text-sm">{pageOf(page, pages)}</li>
        <li>
          {page < pages ? (
            <Link
              className={cn(buttonVariants({ variant: 'ghost' }), 'px-2.5')}
              href={listQueryHref(query, { page: page + 1 })}
              rel="next"
            >
              {PAGER_TEXTS.next}
              <ChevronRightIcon aria-hidden />
            </Link>
          ) : (
            <span className={END}>
              {PAGER_TEXTS.next}
              <ChevronRightIcon className="size-4" aria-hidden />
            </span>
          )}
        </li>
      </ul>
    </nav>
  )
}
