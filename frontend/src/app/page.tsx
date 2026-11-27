import { redirect } from 'next/navigation'

import { rootRedirect } from '@/domain/navigation'
import { getSession } from '@/services/session'

const Page = async () => redirect(rootRedirect(await getSession()))

export default Page
