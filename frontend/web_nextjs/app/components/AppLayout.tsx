'use client';

import React, { ReactNode } from 'react';
import Link from 'next/link';
import { useTheme } from './ThemeProvider';
import { ThemeToggle } from './ThemeToggle';

interface AppLayoutProps {
  children: ReactNode;
  showNav?: boolean;
  title?: string;
}

export function AppLayout({ children, showNav = true, title = 'TigerWallet' }: AppLayoutProps) {
  const { theme, isDark } = useTheme();

  const navLinks = [
    { href: '/wallet', label: 'Wallet' },
    { href: '/swap', label: 'Swap' },
    { href: '/pool', label: 'Pool' },
    { href: '/bridge', label: 'Bridge' },
    { href: '/farming', label: 'Farming' },
    { href: '/portfolio', label: 'Portfolio' },
    { href: '/nft-marketplace', label: 'NFT' },
    { href: '/perpetual', label: 'Perpetual' },
    { href: '/staking', label: 'Staking' },
    { href: '/launchpad', label: 'Launchpad' },
    { href: '/super_admin', label: 'Admin' },
  ];

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900' : 'bg-slate-50'}`}>
      {/* Header */}
      <header className={`sticky top-0 z-50 ${isDark ? 'bg-slate-800/95 border-slate-700' : 'bg-white/95 border-slate-200'} border-b backdrop-blur-sm`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <Link href="/" className="flex items-center gap-2">
              <span className="text-2xl">🐯</span>
              <span className={`text-xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>
                {title}
              </span>
            </Link>

            {/* Navigation */}
            {showNav && (
              <nav className="hidden md:flex items-center gap-1">
                {navLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={`px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isDark
                        ? 'text-slate-300 hover:text-orange-500 hover:bg-slate-700/50'
                        : 'text-slate-600 hover:text-orange-600 hover:bg-slate-100'
                    }`}
                  >
                    {link.label}
                  </Link>
                ))}
              </nav>
            )}

            {/* Actions */}
            <div className="flex items-center gap-3">
              <ThemeToggle />
              <Link
                href="/wallet"
                className={`px-4 py-2 rounded-lg font-medium text-sm ${
                  isDark
                    ? 'bg-orange-500 hover:bg-orange-600 text-white'
                    : 'bg-orange-500 hover:bg-orange-600 text-white'
                }`}
              >
                Connect Wallet
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1">
        {children}
      </main>

      {/* Footer */}
      <footer className={`py-8 border-t ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <div className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
              © 2026 TigerWallet - Enterprise Web3 Wallet
            </div>
            <div className="flex items-center gap-6">
              <Link href="/docs" className={`text-sm ${isDark ? 'text-slate-400 hover:text-orange-500' : 'text-slate-500 hover:text-orange-600'}`}>
                Documentation
              </Link>
              <Link href="/support" className={`text-sm ${isDark ? 'text-slate-400 hover:text-orange-500' : 'text-slate-500 hover:text-orange-600'}`}>
                Support
              </Link>
              <Link href="/terms" className={`text-sm ${isDark ? 'text-slate-400 hover:text-orange-500' : 'text-slate-500 hover:text-orange-600'}`}>
                Terms
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
