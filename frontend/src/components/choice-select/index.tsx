'use client'

import { cn } from 'cn'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectItemText,
  SelectTrigger,
  SelectValue
} from '@/components/ui/shadcn/select'

import type { ChoiceOptionRowProps, ChoiceSelectProps } from './types'

export const ChoiceOptionRow = ({ option }: ChoiceOptionRowProps) => (
  <span className="flex flex-col gap-0.5">
    <SelectItemText
      className={cn(option.risky ? 'text-destructive' : 'text-foreground')}
    >
      {option.label}
    </SelectItemText>
    <span className="text-muted-foreground text-xs whitespace-normal">
      {option.description}
    </span>
  </span>
)

export const ChoiceSelect = ({
  'aria-describedby': describedBy,
  'aria-invalid': invalid,
  'aria-required': required,
  id,
  labelledBy,
  name,
  onBlur,
  onValueChange,
  options,
  placeholder,
  ref,
  value
}: ChoiceSelectProps) => {
  const selected = options.find(option => option.code === value)
  return (
    <Select<string>
      items={options.map(option => ({
        label: option.label,
        value: option.code
      }))}
      onOpenChange={open => {
        if (!open) onBlur()
      }}
      onValueChange={code => {
        if (code) onValueChange(code)
      }}
      name={name}
      value={value === '' ? null : value}
    >
      <SelectTrigger
        aria-describedby={describedBy}
        aria-invalid={invalid}
        aria-labelledby={labelledBy}
        aria-required={required}
        className={cn(selected?.risky && 'text-destructive')}
        id={id}
        ref={ref}
      >
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {options.map(option => (
          <SelectItem key={option.code} value={option.code}>
            <ChoiceOptionRow option={option} />
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
