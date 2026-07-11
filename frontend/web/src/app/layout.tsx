import type { Metadata } from 'next';
import './globals.css';

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
    <html lang="en">
      <body className="bg-dark-950 text-white antialiased">
        <div className="min-h-screen bg-gradient-to-br from-dark-900 via-dark-800 to-dark-900">
          {children}
        </div>
      </body>
    </html>
  );
}
