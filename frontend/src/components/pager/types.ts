import type { ListQuery } from '@/domain/ticket-query/types'

export interface PagerProps {
  pages: number
  query: ListQuery
}
