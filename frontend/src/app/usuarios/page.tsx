import type { Metadata } from 'next'

import { UserProfileScreen } from '@/screens/user-profile'

export const metadata: Metadata = { title: 'Minha conta' }

const Page = async ({ searchParams }: PageProps<'/usuarios'>) => (
  <UserProfileScreen searchParams={await searchParams} />
)

export default Page
