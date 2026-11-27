import { CheckIcon } from 'lucide-react'
import { useId } from 'react'

import { TicketTypeIcon } from '@/components/ticket-type-icon'
import { RadioGroup, RadioGroupItem } from '@/components/ui/shadcn/radio-group'
import type { TicketType } from '@/domain/tickets/types'

import type { TicketTypePickerProps } from './types'

export const TicketTypePicker = ({
  error,
  labelledBy,
  onValueChange,
  options,
  value
}: TicketTypePickerProps) => {
  const id = useId()
  const errorId = `${id}-error`

  const selectOnEnter = (event: React.KeyboardEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement
    if (event.key === 'Enter' && target.getAttribute('role') === 'radio') {
      event.preventDefault()
      target.click()
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <RadioGroup<null | TicketType>
        onValueChange={type => {
          if (type) onValueChange(type)
        }}
        aria-describedby={error ? errorId : undefined}
        aria-invalid={error ? true : undefined}
        aria-labelledby={labelledBy}
        className="grid gap-3 sm:grid-cols-2"
        onKeyDown={selectOnEnter}
        value={value}
      >
        {options.map(option => {
          const labelId = `${id}-${option.code}-label`
          const descriptionId = `${id}-${option.code}-description`
          return (
            <label
              className="border-border hover:bg-muted/50 has-data-checked:border-foreground has-focus-visible:ring-ring/50 group/type relative flex cursor-pointer items-start gap-3 rounded-sm border p-4 transition-colors has-focus-visible:ring-3 motion-reduce:transition-none"
              data-testid={`ticket-type-${option.code}`}
              key={option.code}
            >
              <RadioGroupItem
                aria-describedby={descriptionId}
                aria-labelledby={labelId}
                className="sr-only absolute"
                value={option.code}
              />
              <TicketTypeIcon
                selected={value === option.code}
                type={option.code}
              />
              <span className="flex min-w-0 flex-1 flex-col gap-1">
                <span className="text-sm font-medium" id={labelId}>
                  {option.label}
                </span>
                <span
                  className="text-muted-foreground text-sm"
                  id={descriptionId}
                >
                  {option.description}
                </span>
              </span>
              <CheckIcon
                className="text-foreground hidden size-4 shrink-0 group-has-data-checked/type:block"
                aria-hidden
              />
            </label>
          )
        })}
      </RadioGroup>
      {error ? (
        <p className="text-destructive text-sm" id={errorId}>
          {error}
        </p>
      ) : null}
    </div>
  )
}
