import type { Metadata } from 'next';
import './globals.css';
import { ThemeProvider } from '../context/ThemeContext';

export const metadata: Metadata = {
  title: 'TigerWallet - Enterprise Multi-Chain Wallet',
  description: 'The most advanced decentralized cryptocurrency wallet with 100+ blockchain support, instant swaps, perpetual trading, and copy trading.',
  keywords: ['crypto wallet', 'ethereum', 'bitcoin', 'defi', 'web3', 'trading', 'staking'],
  authors: [{ name: 'TigerWallet' }],
  openGraph: {
    title: 'TigerWallet - Enterprise Multi-Chain Wallet',
    description: 'The most advanced decentralized cryptocurrency wallet',
    type: 'website',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="antialiased" style={{ background: 'var(--bg-primary)', color: 'var(--text-primary)' }}>
        <ThemeProvider>
          <div className="min-h-screen" style={{ background: 'var(--bg-primary)' }}>
            {children}
          </div>
        </ThemeProvider>
      </body>
    </html>
  );
}
