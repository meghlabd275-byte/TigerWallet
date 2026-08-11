import type { Metadata } from 'next'
import { ThemeProvider } from './components/ThemeProvider'
import { AppShell } from './components/AppShell'
import './globals.css'

export const metadata: Metadata = {
  title: 'TigerWallet - Enterprise Web3 Wallet',
  description: 'Enterprise-grade multichain decentralized Web3 wallet with 100+ blockchain support',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (function() {
                // Default to dark theme for crypto wallet
                var theme = localStorage.getItem('tigerwallet_theme_mode');
                if (!theme) {
                  // Check system preference
                  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
                    theme = 'dark';
                  } else {
                    theme = 'light';
                  }
                }
                document.documentElement.classList.add(theme);
                document.documentElement.setAttribute('data-theme', theme);
              })();
            `,
          }}
        />
      </head>
      <body className="antialiased">
        <ThemeProvider>
          <AppShell>{children}</AppShell>
        </ThemeProvider>
      </body>
    </html>
  )
}