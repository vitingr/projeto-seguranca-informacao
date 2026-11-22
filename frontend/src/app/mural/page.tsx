import type { Metadata } from 'next'

import { parsePageParam } from '@/domain/post'
import { PostWall } from '@/screens/post-wall'

export const metadata: Metadata = { title: 'Mural' }

const Page = async ({ searchParams }: PageProps<'/mural'>) => {
  const { pagina } = await searchParams
  return <PostWall page={parsePageParam(pagina)} />
}

export default Page
