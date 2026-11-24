import type { Metadata } from 'next'
import { Geist_Mono, Poppins } from 'next/font/google'
import '@/styles/globals.css'

// next/font self-hosts the files at build time: the browser never calls
// Google, so the fonts stay on the app's own origin.
const poppins = Poppins({
  subsets: ['latin'],
  variable: '--font-poppins',
  weight: ['400', '500', '600', '700']
})

// Technical data only (IPs, times, header and cookie names).
const geistMono = Geist_Mono({
  subsets: ['latin'],
  variable: '--font-geist-mono'
})

export const metadata: Metadata = {
  description: 'Acesso por papéis: administradores e usuários comuns.',
  title: {
    default: 'Sistemas da Informação',
    template: '%s · Sistemas da Informação'
  }
}

export default function RootLayout({ children }: LayoutProps<'/'>) {
  return (
    <html
      className={`${poppins.variable} ${geistMono.variable} h-full antialiased`}
      lang="pt-BR"
    >
      <body className="flex min-h-full flex-col text-neutral-700">
        {children}
      </body>
    </html>
  )
}
