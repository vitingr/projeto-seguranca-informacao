import type { Ref } from 'react'

export interface ReplyBoxProps {
  /** "Fechado": the box and the tray become the closed line (AC-3.8). */
  closed?: boolean
  /** Shown under the closed line ("Novo ticket" for the author). */
  closedAction?: { href: string; label: string }
  /** The field message of an invalid text (AC-3.5). */
  error?: string
  id: string
  /** macOS shows "⌘+Enter" in the tray instead of "Ctrl+Enter". */
  mac?: boolean
  onChange: (value: string) => void
  /** Called on the button and on Ctrl/⌘+Enter, only when there is text. */
  onSend: () => void
  placeholder: string
  /** Sending: the box is read-only and the button shows that it is busy. */
  sending?: boolean
  textareaRef?: Ref<HTMLTextAreaElement>
  /** Who reads the reply, in the tray ("Visível para …", AC-5.4). */
  trayText: string
  value: string
}
