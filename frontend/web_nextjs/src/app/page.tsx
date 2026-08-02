'use client';
import { useState, useEffect } from 'react';

const chains = [
  { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', color: '#627EEA', supported: true },
  { id: 'bsc', name: 'BNB Chain', symbol: 'BNB', color: '#F3BA2F', supported: true },
  { id: 'polygon', name: 'Polygon', symbol: 'MATIC', color: '#8247E5', supported: true },
  { id: 'arbitrum', name: 'Arbitrum', symbol: 'ETH', color: '#28A0F0', supported: true },
  { id: 'optimism', name: 'Optimism', symbol: 'ETH', color: '#FF0420', supported: true },
  { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX', color: '#E84142', supported: true },
  { id: 'solana', name: 'Solana', symbol: 'SOL', color: '#9945FF', supported: true },
  { id: 'base', name: 'Base', symbol: 'ETH', color: '#0052FF', supported: true },
  { id: 'linea', name: 'Linea', symbol: 'ETH', color: '#121212', supported: true },
  { id: 'zksync', name: 'zkSync', symbol: 'ETH', color: '#8B8BEB', supported: true },
  { id: 'tron', name: 'Tron', symbol: 'TRX', color: '#FF0013', supported: true },
  { id: 'cosmos', name: 'Cosmos', symbol: 'ATOM', color: '#2E3148', supported: true },
];

const features = [
  { icon: '🔐', title: 'Secure Wallet', desc: 'Multi-layer encryption with MPC technology' },
  { icon: '⚡', title: 'Instant Swap', desc: 'Best rates across 50+ DEX aggregators' },
  { icon: '🌉', title: 'Cross-Chain', desc: 'Bridge to 100+ blockchains seamlessly' },
  { icon: '🎮', title: 'NFT Support', desc: 'Full NFT gallery and marketplace access' },
  { icon: '📈', title: 'DeFi Staking', desc: 'Stake and earn rewards on multiple chains' },
  { icon: '🔗', title: 'DApp Browser', desc: 'Connect to any Web3 application' },
];

const stats = [
  { value: '500+', label: 'Supported Tokens' },
  { value: '100+', label: 'Blockchain Networks' },
  { value: '$2B+', label: 'Total Value Locked' },
  { value: '1M+', label: 'Active Users' },
];

export default function Home() {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  if (!mounted) return null;

  return (
    <div className="min-h-screen">
      {/* Hero Section */}
      <section className="relative min-h-[90vh] flex items-center justify-center overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-br from-orange-50 via-white to-red-50 dark:from-gray-900 dark:via-gray-950 dark:to-gray-900" />
        <div className="absolute inset-0 opacity-30">
          <div className="absolute top-20 left-10 w-72 h-72 bg-orange-300 rounded-full mix-blend-multiply filter blur-3xl animate-pulse" />
          <div className="absolute top-40 right-10 w-72 h-72 bg-red-300 rounded-full mix-blend-multiply filter blur-3xl animate-pulse" style={{ animationDelay: '1s' }} />
          <div className="absolute bottom-20 left-1/3 w-72 h-72 bg-yellow-300 rounded-full mix-blend-multiply filter blur-3xl animate-pulse" style={{ animationDelay: '2s' }} />
        </div>
        
        <div className="relative z-10 max-w-7xl mx-auto px-4 text-center">
          <div className="inline-flex items-center gap-2 px-4 py-2 bg-orange-100 dark:bg-orange-900/30 rounded-full text-orange-600 dark:text-orange-400 text-sm font-medium mb-8">
            <span className="w-2 h-2 bg-orange-500 rounded-full animate-pulse" />
            Trusted by 1M+ Users Worldwide
          </div>
          
          <h1 className="text-5xl md:text-7xl font-bold mb-6 leading-tight">
            <span className="gradient-text">The Future of</span>
            <br />
            <span>Decentralized Wallet</span>
          </h1>
          
          <p className="text-xl md:text-2xl text-gray-600 dark:text-gray-400 mb-12 max-w-3xl mx-auto">
            Experience seamless Web3 with 100+ blockchain support, instant swaps, and enterprise-grade security. One wallet for everything.
          </p>
          
          <div className="flex flex-col sm:flex-row gap-4 justify-center mb-16">
            <a href="/register" className="btn-primary text-lg px-8 py-4">
              Get Started Free
            </a>
            <a href="/wallet" className="btn-secondary text-lg px-8 py-4">
              Launch Wallet
            </a>
          </div>

          {/* Chain Support */}
          <div className="mb-16">
            <p className="text-sm text-gray-500 mb-4">Supported Blockchains</p>
            <div className="flex flex-wrap justify-center gap-4">
              {chains.map((chain) => (
                <div key={chain.id} className="flex items-center gap-2 px-4 py-2 bg-white/80 dark:bg-gray-800/80 rounded-full shadow-sm backdrop-blur">
                  <div className="w-6 h-6 rounded-full" style={{ backgroundColor: chain.color }} />
                  <span className="text-sm font-medium">{chain.symbol}</span>
                </div>
              ))}
              <div className="flex items-center gap-2 px-4 py-2 bg-white/80 dark:bg-gray-800/80 rounded-full shadow-sm backdrop-blur">
                <span className="text-sm text-gray-500">+90 more</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Stats Section */}
      <section className="py-16 bg-gray-50 dark:bg-gray-900">
        <div className="max-w-7xl mx-auto px-4">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            {stats.map((stat, i) => (
              <div key={i} className="text-center">
                <div className="text-4xl md:text-5xl font-bold gradient-text mb-2">{stat.value}</div>
                <div className="text-gray-600 dark:text-gray-400">{stat.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-24">
        <div className="max-w-7xl mx-auto px-4">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">Everything You Need</h2>
            <p className="text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
              A complete Web3 ecosystem in one powerful wallet
            </p>
          </div>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            {features.map((feature, i) => (
              <div key={i} className="card group hover:-translate-y-1">
                <div className="text-4xl mb-4">{feature.icon}</div>
                <h3 className="text-xl font-semibold mb-2">{feature.title}</h3>
                <p className="text-gray-600 dark:text-gray-400">{feature.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-24 bg-gradient-to-r from-orange-500 to-red-500">
        <div className="max-w-4xl mx-auto px-4 text-center text-white">
          <h2 className="text-4xl font-bold mb-6">Ready to Get Started?</h2>
          <p className="text-xl mb-8 opacity-90">Join millions of users who trust TigerWallet for their digital assets.</p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <a href="/register" className="px-8 py-4 bg-white text-orange-600 rounded-lg font-semibold hover:bg-gray-100 transition-colors">
              Create Free Wallet
            </a>
            <a href="/docs" className="px-8 py-4 bg-white/20 text-white rounded-lg font-semibold hover:bg-white/30 transition-colors">
              Read Documentation
            </a>
          </div>
        </div>
      </section>
    </div>
  );
}
