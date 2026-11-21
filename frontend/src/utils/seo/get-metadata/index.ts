import type { Metadata } from 'next'

import { env } from '@/config/env'

import type { GetMetaDataProps } from './types'

export const getMetaData = ({
  description,
  image,
  title,
  url,
  type = 'website'
}: GetMetaDataProps): Metadata => {
  const metaDataObject: Metadata = {
    title,
    description,
    openGraph: {
      type,
      images: [image],
      title,
      description,
      url
    },
    alternates: {
      canonical: url
    },
    metadataBase: new URL(env.baseUrl),
    twitter: {
      title,
      description,
      images: [image]
    }
  }

  return metaDataObject
}
