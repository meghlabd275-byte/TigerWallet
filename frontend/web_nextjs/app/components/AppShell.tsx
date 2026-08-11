'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { ThemeToggle } from './ThemeToggle'

const NAV = [
  { href: '/wallet', label: 'Wallet' },
  { href: '/swap', label: 'Swap' },
  { href: '/bridge', label: 'Bridge' },
  { href: '/portfolio', label: 'Portfolio' },
  { href: '/nft', label: 'NFTs' },
  { href: '/staking', label: 'Staking' },
  { href: '/dapp-browser', label: 'dApps' },
]

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()

  return (
    <div className="min-h-screen flex flex-col" style={{ background: 'var(--bg-primary)', color: 'var(--text-primary)' }}>
      <header
        className="flex items-center justify-between px-6 py-3 sticky top-0 z-50"
        style={{ background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-color)' }}
      >
        <Link href="/" className="flex items-center gap-2 font-bold text-lg" style={{ color: 'var(--accent)' }}>
          🐯 TigerWallet
        </Link>
        <nav className="hidden md:flex items-center gap-1">
          {NAV.map((item) => {
            const active = pathname === item.href || pathname?.startsWith(item.href + '/')
            return (
              <Link
                key={item.href}
                href={item.href}
                className="px-3 py-1.5 rounded-md text-sm transition-colors"
                style={{
                  color: active ? 'var(--accent)' : 'var(--text-secondary)',
                  background: active ? 'var(--bg-tertiary)' : 'transparent',
                }}
              >
                {item.label}
              </Link>
            )
          })}
        </nav>
        <ThemeToggle />
      </header>
      <main className="flex-1">{children}</main>
    </div>
  )
}
