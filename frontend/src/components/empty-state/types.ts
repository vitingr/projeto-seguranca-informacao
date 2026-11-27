export interface EmptyStateProps {
  /** The one useful next action (spec section 7, "Empty"). */
  action?: { href: string; label: string }
  text: string
}
