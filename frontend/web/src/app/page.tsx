'use client';

import { useState, useEffect } from 'react';
import Head from 'next/head';
import Link from 'next/link';
import { 
  Wallet, 
  Shield, 
  Zap, 
  Globe, 
  ArrowRight, 
  Copy, 
  TrendingUp, 
  Lock, 
  Users, 
  Coins, 
  ArrowDownUp,
  LineChart,
  ChevronRight,
  CheckCircle,
  Star,
  Menu,
  X,
  WalletCards,
  Smartphone,
  Globe2,
  CreditCard,
  Hexagon
} from 'lucide-react';

// Feature data
const features = [
  {
    icon: Globe,
    title: '100+ Blockchains',
    description: 'Support for all major EVM, Solana, Cosmos, Aptos, and 100+ networks',
  },
  {
    icon: ArrowDownUp,
    title: 'Instant Swaps',
    description: 'Lightning-fast token swaps with best price routing across DEXs',
  },
  {
    icon: LineChart,
    title: 'Perpetual Trading',
    description: 'Trade futures with up to 100x leverage on 50+ markets',
  },
  {
    icon: Users,
    title: 'Copy Trading',
    description: 'Follow top traders and automatically copy their strategies',
  },
  {
    icon: Coins,
    title: 'Staking',
    description: 'Earn rewards by staking 50+ tokens with competitive APY',
  },
  {
    icon: Shield,
    title: 'Bank-Grade Security',
    description: 'MPC encryption, hardware wallet support, and multi-sig protection',
  },
];

// Supported blockchains
const blockchains = [
  { name: 'Ethereum', symbol: 'ETH', color: '#627EEA' },
  { name: 'BNB Chain', symbol: 'BNB', color: '#F3BA2F' },
  { name: 'Solana', symbol: 'SOL', color: '#9945FF' },
  { name: 'Polygon', symbol: 'MATIC', color: '#8247E5' },
  { name: 'Arbitrum', symbol: 'ETH', color: '#28A0F0' },
  { name: 'Optimism', symbol: 'OP', color: '#FF0420' },
  { name: 'Base', symbol: 'BASE', color: '#0052FF' },
  { name: 'Avalanche', symbol: 'AVAX', color: '#E84142' },
  { name: 'TRON', symbol: 'TRX', color: '#FF0013' },
  { name: 'Toncoin', symbol: 'TON', color: '#0098EA' },
  { name: 'Aptos', symbol: 'APT', color: '#000000' },
  { name: 'Sui', symbol: 'SUI', color: '#6FBEF5' },
  { name: 'Cosmos', symbol: 'ATOM', color: '#2E3148' },
  { name: 'NEAR', symbol: 'NEAR', color: '#00C08B' },
  { name: 'Polkadot', symbol: 'DOT', color: '#E6007A' },
];

// Stats
const stats = [
  { value: '100+', label: 'Blockchains' },
  { value: '200+', label: 'Tokens' },
  { value: '$0', label: 'Transaction Fees*' },
  { value: '99.9%', label: 'Uptime' },
];

export default function Home() {
  const [mounted, setMounted] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted) return null;

  return (
    <>
      <Head>
        <title>TigerWallet - Enterprise Multi-Chain Wallet</title>
        <meta name="description" content="The most advanced decentralized cryptocurrency wallet with 100+ blockchain support" />
      </Head>

      {/* Header */}
      <header className="fixed top-0 left-0 right-0 z-50 glass-dark">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <div className="flex items-center gap-2">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
                <Hexagon className="w-6 h-6 text-white" />
              </div>
              <span className="text-xl font-bold gradient-text">TigerWallet</span>
            </div>

            {/* Desktop Navigation */}
            <nav className="hidden md:flex items-center gap-8">
              <a href="#features" className="text-gray-300 hover:text-white transition-colors">Features</a>
              <a href="#blockchains" className="text-gray-300 hover:text-white transition-colors">Blockchains</a>
              <a href="#stats" className="text-gray-300 hover:text-white transition-colors">Stats</a>
              <a href="#roadmap" className="text-gray-300 hover:text-white transition-colors">Roadmap</a>
            </nav>

            {/* CTA Buttons */}
            <div className="hidden md:flex items-center gap-4">
              <button className="text-gray-300 hover:text-white transition-colors">
                Connect
              </button>
              <button className="px-5 py-2 bg-gradient-to-r from-primary-500 to-primary-600 rounded-lg font-medium hover:from-primary-600 hover:to-primary-700 transition-all glow-hover">
                Download App
              </button>
            </div>

            {/* Mobile Menu Button */}
            <button 
              className="md:hidden p-2 text-gray-300"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            >
              {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>
        </div>

        {/* Mobile Menu */}
        {mobileMenuOpen && (
          <div className="md:hidden glass-dark border-t border-white/10">
            <div className="px-4 py-4 space-y-3">
              <a href="#features" className="block text-gray-300 hover:text-white">Features</a>
              <a href="#blockchains" className="block text-gray-300 hover:text-white">Blockchains</a>
              <a href="#stats" className="block text-gray-300 hover:text-white">Stats</a>
              <button className="w-full py-3 bg-gradient-to-r from-primary-500 to-primary-600 rounded-lg font-medium">
                Download App
              </button>
            </div>
          </div>
        )}
      </header>

      {/* Hero Section */}
      <section className="relative pt-32 pb-20 overflow-hidden">
        {/* Background Effects */}
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-primary-500/20 rounded-full blur-3xl animate-pulse-slow" />
          <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-primary-700/20 rounded-full blur-3xl animate-pulse-slow" style={{ animationDelay: '1s' }} />
        </div>

        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center">
            {/* Badge */}
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full glass mb-8 animate-slide-up">
              <Star className="w-4 h-4 text-primary-500" />
              <span className="text-sm text-gray-300">Trusted by 1M+ Users Worldwide</span>
            </div>

            {/* Headline */}
            <h1 className="text-5xl md:text-7xl font-bold mb-6 animate-slide-up animation-delay-100">
              <span className="text-white">The Future of</span>
              <br />
              <span className="gradient-text">Decentralized Wallet</span>
            </h1>

            {/* Subheadline */}
            <p className="text-xl md:text-2xl text-gray-400 max-w-3xl mx-auto mb-10 animate-slide-up animation-delay-200">
              Experience seamless DeFi with 100+ blockchains, instant swaps, perpetual trading, 
              and copy trading - all in one secure, non-custodial wallet.
            </p>

            {/* CTA Buttons */}
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 animate-slide-up animation-delay-300">
              <button className="w-full sm:w-auto px-8 py-4 bg-gradient-to-r from-primary-500 to-primary-600 rounded-xl font-bold text-lg hover:from-primary-600 hover:to-primary-700 transition-all glow flex items-center justify-center gap-2">
                <WalletCards className="w-5 h-5" />
                Create Wallet
                <ArrowRight className="w-5 h-5" />
              </button>
              <button className="w-full sm:w-auto px-8 py-4 glass rounded-xl font-bold text-lg hover:bg-white/10 transition-all flex items-center justify-center gap-2">
                <Globe2 className="w-5 h-5" />
                View Supported Chains
              </button>
            </div>

            {/* Stats Row */}
            <div id="stats" className="mt-16 grid grid-cols-2 md:grid-cols-4 gap-8 animate-slide-up animation-delay-400">
              {stats.map((stat, index) => (
                <div key={index} className="text-center">
                  <div className="text-3xl md:text-4xl font-bold gradient-text">{stat.value}</div>
                  <div className="text-gray-500 mt-1">{stat.label}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Wallet Preview */}
          <div className="mt-20 relative animate-slide-up animation-delay-500">
            <div className="relative mx-auto max-w-md">
              {/* Phone Mockup */}
              <div className="relative mx-auto max-w-sm">
                <div className="rounded-[3rem] overflow-hidden border-4 border-dark-700 shadow-2xl">
                  <div className="bg-dark-900 p-6" style={{ height: '600px' }}>
                    {/* Status Bar */}
                    <div className="flex justify-between items-center mb-6">
                      <div className="text-sm text-gray-400">9:41</div>
                      <div className="flex gap-2">
                        <div className="w-4 h-4 rounded-full bg-primary-500" />
                        <div className="w-4 h-4 rounded-full bg-gray-600" />
                      </div>
                    </div>
                    
                    {/* Balance */}
                    <div className="text-center mb-6">
                      <div className="text-sm text-gray-500 mb-1">Total Balance</div>
                      <div className="text-4xl font-bold text-white">$124,592.84</div>
                    </div>

                    {/* Action Buttons */}
                    <div className="grid grid-cols-4 gap-2 mb-8">
                      {[
                        { icon: ArrowDownUp, label: 'Swap' },
                        { icon: ArrowRight, label: 'Send' },
                        { icon: ArrowRight, label: 'Receive' },
                        { icon: LineChart, label: 'Trade' },
                      ].map((action, i) => (
                        <button key={i} className="flex flex-col items-center gap-2 p-3 rounded-xl hover:bg-white/5 transition-colors">
                          <div className="w-12 h-12 rounded-full bg-primary-500/20 flex items-center justify-center">
                            <action.icon className="w-5 h-5 text-primary-500" />
                          </div>
                          <span className="text-xs text-gray-400">{action.label}</span>
                        </button>
                      ))}
                    </div>

                    {/* Assets List */}
                    <div className="space-y-3">
                      {[
                        { symbol: 'ETH', name: 'Ethereum', balance: '32.5', value: '$112,125', color: '#627EEA' },
                        { symbol: 'BTC', name: 'Bitcoin', balance: '0.185', value: '$12,485', color: '#F7931A' },
                        { symbol: 'SOL', name: 'Solana', balance: '15', value: '$2,175', color: '#9945FF' },
                      ].map((asset, i) => (
                        <div key={i} className="flex items-center justify-between p-3 rounded-xl glass">
                          <div className="flex items-center gap-3">
                            <div className="w-10 h-10 rounded-full flex items-center justify-center text-sm font-bold" style={{ backgroundColor: `${asset.color}30`, color: asset.color }}>
                              {asset.symbol.slice(0, 2)}
                            </div>
                            <div>
                              <div className="font-medium">{asset.symbol}</div>
                              <div className="text-xs text-gray-500">{asset.name}</div>
                            </div>
                          </div>
                          <div className="text-right">
                            <div className="font-medium">{asset.balance}</div>
                            <div className="text-xs text-gray-500">{asset.value}</div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 bg-dark-900/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">
              <span className="gradient-text">Everything You Need</span> in One Wallet
            </h2>
            <p className="text-xl text-gray-400 max-w-2xl mx-auto">
              From simple transfers to advanced DeFi trading, TigerWallet has you covered with enterprise-grade features.
            </p>
          </div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {features.map((feature, index) => (
              <div 
                key={index}
                className="group p-6 rounded-2xl glass hover:bg-white/10 transition-all hover:scale-105"
                style={{ animationDelay: `${index * 100}ms` }}
              >
                <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-primary-500/20 to-primary-700/20 flex items-center justify-center mb-4 group-hover:from-primary-500/30 group-hover:to-primary-700/30 transition-all">
                  <feature.icon className="w-7 h-7 text-primary-500" />
                </div>
                <h3 className="text-xl font-bold mb-2">{feature.title}</h3>
                <p className="text-gray-400">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Blockchains Section */}
      <section id="blockchains" className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">
              <span className="gradient-text">100+ Blockchains</span> Supported
            </h2>
            <p className="text-xl text-gray-400 max-w-2xl mx-auto">
              From Ethereum to emerging chains, manage all your assets in one place.
            </p>
          </div>

          <div className="flex flex-wrap justify-center gap-4">
            {blockchains.map((chain, index) => (
              <div 
                key={index}
                className="flex items-center gap-3 px-5 py-3 rounded-full glass hover:bg-white/10 transition-all cursor-pointer"
              >
                <div 
                  className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold"
                  style={{ backgroundColor: `${chain.color}30`, color: chain.color }}
                >
                  {chain.symbol.slice(0, 2)}
                </div>
                <span className="font-medium">{chain.name}</span>
              </div>
            ))}
          </div>

          <div className="text-center mt-10">
            <button className="text-primary-500 hover:text-primary-400 flex items-center gap-2 mx-auto">
              View all 100+ blockchains <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </section>

      {/* Security Section */}
      <section className="py-20 bg-dark-900/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <div>
              <h2 className="text-4xl font-bold mb-6">
                <span className="gradient-text">Bank-Grade Security</span> for Your Assets
              </h2>
              <p className="text-xl text-gray-400 mb-8">
                Your keys, your crypto. TigerWallet uses industry-leading security to keep your assets safe.
              </p>
              
              <div className="space-y-4">
                {[
                  { icon: Lock, title: 'Non-Custodial', desc: 'You control your private keys' },
                  { icon: Shield, title: 'MPC Encryption', desc: 'Multi-party computation security' },
                  { icon: Smartphone, title: 'Hardware Wallet', desc: 'Ledger & Keystone support' },
                  { icon: CheckCircle, title: 'Audited Code', desc: 'Regular security audits' },
                ].map((item, index) => (
                  <div key={index} className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-lg bg-primary-500/20 flex items-center justify-center flex-shrink-0">
                      <item.icon className="w-5 h-5 text-primary-500" />
                    </div>
                    <div>
                      <div className="font-semibold">{item.title}</div>
                      <div className="text-sm text-gray-500">{item.desc}</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="relative">
              <div className="absolute inset-0 bg-gradient-to-r from-primary-500/20 to-transparent rounded-3xl blur-3xl" />
              <div className="relative glass rounded-3xl p-8">
                <div className="space-y-4">
                  <div className="flex justify-between items-center p-4 rounded-xl bg-dark-800/50">
                    <span className="text-gray-400">Encryption</span>
                    <CheckCircle className="w-5 h-5 text-green-500" />
                  </div>
                  <div className="flex justify-between items-center p-4 rounded-xl bg-dark-800/50">
                    <span className="text-gray-400">2FA Authentication</span>
                    <CheckCircle className="w-5 h-5 text-green-500" />
                  </div>
                  <div className="flex justify-between items-center p-4 rounded-xl bg-dark-800/50">
                    <span className="text-gray-400">Biometric Lock</span>
                    <CheckCircle className="w-5 h-5 text-green-500" />
                  </div>
                  <div className="flex justify-between items-center p-4 rounded-xl bg-dark-800/50">
                    <span className="text-gray-400">Withdrawal Whitelist</span>
                    <CheckCircle className="w-5 h-5 text-green-500" />
                  </div>
                  <div className="flex justify-between items-center p-4 rounded-xl bg-dark-800/50">
                    <span className="text-gray-400">Transaction Limits</span>
                    <CheckCircle className="w-5 h-5 text-green-500" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="relative rounded-3xl overflow-hidden">
            <div className="absolute inset-0 bg-gradient-to-r from-primary-600 to-primary-800" />
            <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHZpZXdCb3g9IjAgMCA2MCA2MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48ZyBmaWxsPSJub25lIiBmaWxsLXJ1bGU9ImV2ZW5vZGQiPjxwYXRoIGQ9Ik0zNiAxOGMtOS45NDEgMC0xOCA4LjA1OS0xOCAxOHM4LjA1OSAxOCAxOCAxOCAxOC04LjA1OSAxOC0xOC04LjA1OS0xOC0xOC0xOHptMCAzMmMtNy43MzIgMC0xNC02LjI2OC0xNC0xNHM2LjI2OC0xNCAxNC0xNCAxNCA2LjI2OCAxNCAxNC02LjI2OCAxNC0xNCAxNHoiIGZpbGw9IiNmZmYiIGZpbGwtb3BhY2l0eT0iLjEiLz48L2c+PC9zdmc+')] opacity-30" />
            <div className="relative px-8 py-16 text-center">
              <h2 className="text-4xl font-bold mb-4">Ready to Get Started?</h2>
              <p className="text-xl text-primary-100 mb-8 max-w-xl mx-auto">
                Download TigerWallet today and experience the future of decentralized finance.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <button className="w-full sm:w-auto px-8 py-4 bg-white text-primary-600 rounded-xl font-bold hover:bg-gray-100 transition-all flex items-center justify-center gap-2">
                  <Wallet className="w-5 h-5" />
                  Create Free Wallet
                </button>
                <button className="w-full sm:w-auto px-8 py-4 border-2 border-white/30 rounded-xl font-bold hover:bg-white/10 transition-all flex items-center justify-center gap-2">
                  <Copy className="w-5 h-5" />
                  Import Existing
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-12 border-t border-white/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid md:grid-cols-4 gap-8 mb-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
                  <Hexagon className="w-5 h-5 text-white" />
                </div>
                <span className="text-lg font-bold">TigerWallet</span>
              </div>
              <p className="text-gray-500 text-sm">
                The most advanced decentralized cryptocurrency wallet for the future of finance.
              </p>
            </div>
            
            <div>
              <h4 className="font-semibold mb-4">Product</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="#" className="hover:text-white">Features</a></li>
                <li><a href="#" className="hover:text-white">Security</a></li>
                <li><a href="#" className="hover:text-white">Pricing</a></li>
                <li><a href="#" className="hover:text-white">API</a></li>
              </ul>
            </div>
            
            <div>
              <h4 className="font-semibold mb-4">Resources</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="#" className="hover:text-white">Documentation</a></li>
                <li><a href="#" className="hover:text-white">Support</a></li>
                <li><a href="#" className="hover:text-white">Blog</a></li>
                <li><a href="#" className="hover:text-white">Status</a></li>
              </ul>
            </div>
            
            <div>
              <h4 className="font-semibold mb-4">Legal</h4>
              <ul className="space-y-2 text-sm text-gray-500">
                <li><a href="#" className="hover:text-white">Privacy</a></li>
                <li><a href="#" className="hover:text-white">Terms</a></li>
                <li><a href="#" className="hover:text-white">Cookies</a></li>
              </ul>
            </div>
          </div>
          
          <div className="pt-8 border-t border-white/10 flex flex-col md:flex-row justify-between items-center gap-4">
            <div className="text-sm text-gray-500">
              © 2026 TigerWallet. All rights reserved.
            </div>
            <div className="flex gap-4">
              {/* Social Links */}
            </div>
          </div>
        </div>
      </footer>
    </>
  );
}
