import { cn } from 'cn'
import { ArrowUpRight } from 'lucide-react'
import Image from 'next/image'
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
      'group bg-card hover:border-foreground/25 has-focus-visible:border-ring has-focus-visible:ring-ring/50 relative flex h-full flex-col rounded-sm border has-focus-visible:ring-3 motion-safe:transition-[border-color,translate] motion-safe:duration-300',
      className
    )}
  >
    <figure className="max-h-45 w-full overflow-hidden">
      <Image
        alt="Login Background Image"
        className="h-full w-full object-cover"
        fetchPriority="high"
        height={1920}
        loading="eager"
        src="/login-background.png"
        width={1920}
      />
    </figure>
    <div className="flex flex-col gap-6 px-4 py-5">
      <div className="flex items-start justify-between gap-4">
        <span
          aria-hidden="true"
          className="flex size-5 items-center justify-center rounded-sm text-orange-400 motion-safe:transition-colors motion-safe:duration-300"
        >
          <Icon className="size-4.5" strokeWidth={1.75} />
        </span>
        <ArrowUpRight
          aria-hidden="true"
          className="text-muted-foreground group-hover:text-foreground size-5"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <span className="text-muted-foreground text-xs font-medium tracking-wider uppercase">
          {kicker}
        </span>
        <h2 className="text-lg font-semibold tracking-tight">
          <Link
            className="outline-none after:absolute after:inset-0 after:rounded-sm"
            href={href}
          >
            {label}
          </Link>
        </h2>
        <p className="text-muted-foreground text-xs">{description}</p>
      </div>
    </div>
  </article>
)
