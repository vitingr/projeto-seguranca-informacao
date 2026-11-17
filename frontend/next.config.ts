import type { NextConfig } from 'next'

import { parseApiUrl } from './src/config/api'

// Throws at startup when API_URL is not an https origin.
const apiUrl = parseApiUrl(process.env.API_URL)

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
  // The browser calls /api/* on the frontend origin; Next forwards it to the
  // backend, so cookies and Fetch Metadata see a single origin.
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${apiUrl}/api/:path*` }]
  }
}

export default nextConfig
