import { cn } from 'cn'

import { mergeProps } from '@base-ui/react/merge-props'
import { useRender } from '@base-ui/react/use-render'

import type { BadgeProps } from './types'
import { badgeVariants } from './variance'

function Badge({
  className,
  render,
  variant = 'default',
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
