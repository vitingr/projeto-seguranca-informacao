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

/**
 * The text of an option in the menu, as the menu rows of
 * `design/references/input-texto.mp4`: the label, and a muted description
 * under it. A risky option ("Produção", "Crítica") has its label in
 * discreet red, like "Full access" in the video.
 */
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

/**
 * A fixed-choice field as a Select (spec Q-17, replaces plan P-2): the
 * trigger shows the chosen label, and the menu lists each option with its
 * description and a check on the selected one.
 */
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
