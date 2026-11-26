import { CircleAlertIcon } from 'lucide-react'

import { Alert, AlertDescription } from '@/components/ui/shadcn/alert'

import type { InlineAlertProps } from './types'

/**
 * A sending error above the actions (spec section 7 "Error"): a thin red
 * border, announced politely (`role="status"`), never a browser dialog.
 */
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
