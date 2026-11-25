import { cn } from 'cn'

import { mergeProps } from '@base-ui/react/merge-props'
import { useRender } from '@base-ui/react/use-render'

import type { BadgeProps } from './types'
import { badgeVariants } from './variance'

// shadcn badge (base-nova), adapted to the design rules: radius `sm`, no
// dark variants.

function Badge({
  className,
  variant = 'default',
  render,
  ...props
}: BadgeProps) {
  return useRender({
    defaultTagName: 'span',
    props: mergeProps<'span'>(
      {
        className: cn(badgeVariants({ variant }), className)
      },
      props
    ),
    render,
    state: {
      slot: 'badge',
      variant
    }
  })
}

export { Badge, badgeVariants }
