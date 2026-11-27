import type { Ref } from 'react'

export interface ReplyBoxProps {
  closed?: boolean
  closedAction?: { href: string; label: string }
  error?: string
  id: string
  mac?: boolean
  onChange: (value: string) => void
  onSend: () => void
  placeholder: string
  sending?: boolean
  textareaRef?: Ref<HTMLTextAreaElement>
  trayText: string
  value: string
}
