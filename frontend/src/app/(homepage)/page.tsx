import { PublicHome, RoleHome } from '@/screens/home'
import { getSession } from '@/services/session'

// Public page (spec D-10). Without a session cookie it does not call the
// API; with one, it shows the links of the user's role.
const Page = async () => {
  const session = await getSession()

  if (session.kind !== 'user') {
    return <PublicHome />
  }

  return <RoleHome role={session.user.role} />
}

export default Page
