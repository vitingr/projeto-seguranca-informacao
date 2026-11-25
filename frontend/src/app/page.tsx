import { redirect } from 'next/navigation'

import { rootRedirect } from '@/domain/navigation'
import { getSession } from '@/services/session'

// There is no public home (spec AC-0.6): `/` only redirects. Without a
// session cookie it does not call the API.
const Page = async () => redirect(rootRedirect(await getSession()))

export default Page
