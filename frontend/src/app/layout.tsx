import type { Metadata } from 'next'
import { Poppins } from 'next/font/google'
import '@/styles/globals.css'

const poppins = Poppins({
  style: ['normal'],
  subsets: ['latin', 'latin-ext'],
  variable: '--font-poppins',
  weight: ['400', '500', '600', '700']
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
    <html className={`${poppins.variable} h-full antialiased`} lang="pt-BR">
      <body className="flex min-h-full flex-col">{children}</body>
    </html>
  )
}
