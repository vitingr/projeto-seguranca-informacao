import { CircleAlertIcon } from 'lucide-react'

import { Alert, AlertDescription } from '@/components/ui/shadcn/alert'

import type { InlineAlertProps } from './types'

export const InlineAlert = ({ children }: InlineAlertProps) => (
  <Alert
    className="border-destructive/40"
    data-testid="send-error"
    role="status"
    variant="destructive"
  >
    <CircleAlertIcon aria-hidden />
    <AlertDescription>{children}</AlertDescription>
  </Alert>
)
