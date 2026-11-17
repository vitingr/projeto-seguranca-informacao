import type { Metadata } from 'next'
import { Geist, Geist_Mono } from 'next/font/google'
import './globals.css'

const geistSans = Geist({
  variable: '--font-geist-sans',
  subsets: ['latin']
})

const geistMono = Geist_Mono({
  variable: '--font-geist-mono',
  subsets: ['latin']
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
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      lang="pt-BR"
    >
      <body className="flex min-h-full flex-col">{children}</body>
    </html>
  )
}
