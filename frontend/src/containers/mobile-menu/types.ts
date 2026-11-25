import type { ProtectedRoute } from '@/domain/protected-routes/types'

export interface MobileMenuProps {
  links: readonly { href: ProtectedRoute; label: string }[]
  roleName: string
}
