import type { NextConfig } from 'next'

import { parseApiUrl } from './src/config/api'

const apiUrl = parseApiUrl({ raw: process.env.API_URL })

const nextConfig: NextConfig = {
  reactCompiler: true,
  turbopack: {
    rules: {
      '*.css': {
        loaders: ['@tailwindcss/turbopack'],
        as: '*.css'
      }
    }
  },
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${apiUrl}/api/:path*` }]
  }
}

export default nextConfig
