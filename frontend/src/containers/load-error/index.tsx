'use client'

import { CircleAlertIcon } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useTransition } from 'react'

import { Alert, AlertDescription } from '@/components/ui/shadcn/alert'
import { Button } from '@/components/ui/shadcn/button'

import type { LoadErrorProps } from './types'

export const LOAD_ERROR_TEXTS = {
  retry: 'Tentar novamente',
  retrying: 'Tentando…'
} as const

/**
 * A list or a ticket that could not be loaded (spec 003 AC-1.10, plan 6.8):
 * an inline alert in place of the content, and "Tentar novamente", which
 * renders the page again on the server (`router.refresh()`). Nothing from
 * an earlier answer is kept on screen.
 */
export const LoadError = ({ message }: LoadErrorProps) => {
  const router = useRouter()
  const [pending, startTransition] = useTransition()
  return (
    <Alert
      className="border-destructive/40 flex flex-col gap-3 px-4 py-3"
      data-testid="load-error"
      variant="destructive"
    >
      <div className="flex items-start gap-2">
        <CircleAlertIcon className="size-4 shrink-0" aria-hidden />
        <AlertDescription className="text-destructive">
          {message}
        </AlertDescription>
      </div>
      <Button
        onClick={() => {
          if (pending) return
          startTransition(() => {
            router.refresh()
          })
        }}
        aria-disabled={pending || undefined}
        className="w-fit"
        type="button"
        variant="outline"
      >
        {pending ? LOAD_ERROR_TEXTS.retrying : LOAD_ERROR_TEXTS.retry}
      </Button>
    </Alert>
  )
}
