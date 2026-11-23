import type { Metadata } from 'next'

import { parseAuditTypeParam } from '@/domain/audit'
import { parsePageParam } from '@/domain/post'
import { AuditLog } from '@/screens/audit-log'

export const metadata: Metadata = { title: 'Auditoria' }

const Page = async ({ searchParams }: PageProps<'/admin/auditoria'>) => {
  const { pagina, tipo } = await searchParams
  return (
    <AuditLog page={parsePageParam(pagina)} type={parseAuditTypeParam(tipo)} />
  )
}

export default Page
