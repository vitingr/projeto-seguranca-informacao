import type { VariantProps } from 'class-variance-authority'
import type { ComponentProps } from 'react'

import type { alertVariants } from './variance'

export type AlertProps = ComponentProps<'div'> &
  VariantProps<typeof alertVariants>

export type AlertPartProps = ComponentProps<'div'>
