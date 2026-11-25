import type { Select as SelectPrimitive } from '@base-ui/react/select'

export type SelectTriggerProps = SelectPrimitive.Trigger.Props & {
  size?: 'default' | 'sm'
}

export type SelectContentProps = SelectPrimitive.Popup.Props &
  Pick<
    SelectPrimitive.Positioner.Props,
    'align' | 'alignItemWithTrigger' | 'alignOffset' | 'side' | 'sideOffset'
  >

export type SelectItemProps = SelectPrimitive.Item.Props
