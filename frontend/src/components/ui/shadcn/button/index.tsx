import { cn } from 'cn'

import { Button as ButtonPrimitive } from '@base-ui/react/button'

import type { ButtonProps } from './types'
import { buttonVariants } from './variance'

const Button = ({
  className,
  variant = 'default',
  size = 'default',
  ...props
}: ButtonProps) => {
  return (
    <ButtonPrimitive
      className={cn(buttonVariants({ variant, size, className }))}
      data-slot="button"
      {...props}
    />
  )
}

export { Button, buttonVariants }
