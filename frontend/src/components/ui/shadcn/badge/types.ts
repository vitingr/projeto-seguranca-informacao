import type { VariantProps } from 'class-variance-authority'

import type { useRender } from '@base-ui/react/use-render'

import type { badgeVariants } from './variance'

export type BadgeProps = useRender.ComponentProps<'span'> &
  VariantProps<typeof badgeVariants>
