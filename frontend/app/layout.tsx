import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'ThreatScan - Cybersecurity Dashboard',
  description: 'Modern file scanning and threat detection dashboard',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className="dark">
      <body className={inter.className}>
        <div className="min-h-screen bg-gradient-to-br from-cyber via-cyber-dark to-cyber-dark">
          <nav className="border-b border-slate-700/50 bg-cyber-dark/50 backdrop-blur">
            <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 bg-gradient-to-br from-safe to-blue-500 rounded-lg"></div>
                <h1 className="text-xl font-bold">ThreatScan</h1>
              </div>
              <div className="flex gap-6">
                <a href="/" className="hover:text-safe transition">Dashboard</a>
                <a href="/upload" className="hover:text-safe transition">Scan</a>
                <a href="/history" className="hover:text-safe transition">History</a>
              </div>
            </div>
          </nav>
          <main className="max-w-7xl mx-auto px-4 py-8">
            {children}
          </main>
        </div>
      </body>
    </html>
  )
}
