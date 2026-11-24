import { cn } from 'cn'
import { ArrowUpRight } from 'lucide-react'
import Link from 'next/link'

import type { AreaCardProps } from './types'

// The whole card is clickable through the link's ::after; the link text
// stays the area label alone.
export const AreaCard: React.FC<AreaCardProps> = ({
  className,
  description,
  href,
  icon: Icon,
  kicker,
  label
}) => (
  <article
    className={cn(
      'group bg-card hover:border-foreground/25 has-focus-visible:border-ring has-focus-visible:ring-ring/50 relative flex h-full flex-col gap-8 rounded-2xl border p-5 has-focus-visible:ring-3 motion-safe:transition-[border-color,translate] motion-safe:duration-300 motion-safe:hover:-translate-y-1',
      className
    )}
  >
    <div className="flex items-start justify-between gap-4">
      <span
        aria-hidden="true"
        className="bg-brand-soft text-brand-ink group-hover:bg-brand group-hover:text-brand-foreground flex size-11 items-center justify-center rounded-xl motion-safe:transition-colors motion-safe:duration-300"
      >
        <Icon className="size-5" strokeWidth={1.75} />
      </span>
      <ArrowUpRight
        aria-hidden="true"
        className="text-muted-foreground group-hover:text-foreground size-5 motion-safe:transition-[color,translate] motion-safe:duration-300 motion-safe:group-hover:translate-x-0.5 motion-safe:group-hover:-translate-y-0.5"
      />
    </div>
    <div className="flex flex-col gap-1.5">
      <span className="text-muted-foreground text-xs font-medium tracking-wider uppercase">
        {kicker}
      </span>
      <h2 className="text-lg font-semibold tracking-tight">
        <Link
          className="outline-none after:absolute after:inset-0 after:rounded-2xl"
          href={href}
        >
          {label}
        </Link>
      </h2>
      <p className="text-muted-foreground text-sm text-pretty">{description}</p>
    </div>
  </article>
)
