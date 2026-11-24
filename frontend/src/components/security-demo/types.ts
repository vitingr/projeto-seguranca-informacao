import type { BadgeProps } from '@/components/ui/shadcn/badge/types'

export interface DemoAuditRowProps {
  className: string
  detail: string
  event: string
  time: string
  variant: BadgeProps['variant']
  who: string
}

export interface DemoPaneProps {
  children: React.ReactNode
  icon: React.ReactNode
  title: string
}
