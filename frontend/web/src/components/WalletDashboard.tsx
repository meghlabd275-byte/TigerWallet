'use client';

import React, { useState, useEffect } from 'react';
import { Wallet, TrendingUp, ArrowUpRight, ArrowDownRight, Activity, Coins, Shield, Globe, Zap } from 'lucide-react';
import { SUPPORTED_BLOCKCHAINS } from '@/types/blockchain';

interface WalletDashboardProps {
  address: string;
  balances: Array<{
    symbol: string;
    name: string;
    balance: string;
    value: number;
    change24h: number;
    logo?: string;
  }>;
}

export default function WalletDashboard({ address, balances }: WalletDashboardProps) {
  const [totalValue, setTotalValue] = useState(0);
  const [change24h, setChange24h] = useState(0);

  useEffect(() => {
    const total = balances.reduce((sum, b) => sum + b.value, 0);
    setTotalValue(total);
    const avgChange = balances.length > 0 
      ? balances.reduce((sum, b) => sum + b.change24h, 0) / balances.length 
      : 0;
    setChange24h(avgChange);
  }, [balances]);

  const formatAddress = (addr: string) => {
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
  };

  const formatValue = (value: number) => {
    if (value >= 1000000) return `$${(value / 1000000).toFixed(2)}M`;
    if (value >= 1000) return `$${(value / 1000).toFixed(2)}K`;
    return `$${value.toFixed(2)}`;
  };

  return (
    <div className="min-h-screen bg-dark-950">
      {/* Header */}
      <header className="glass-dark border-b border-white/10">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
              <Wallet className="w-6 h-6 text-white" />
            </div>
            <span className="text-xl font-bold gradient-text">TigerWallet</span>
          </div>
          <div className="flex items-center gap-4">
            <button className="p-2 rounded-lg hover:bg-white/10 transition-colors">
              <Shield className="w-5 h-5 text-gray-400" />
            </button>
            <button className="p-2 rounded-lg hover:bg-white/10 transition-colors">
              <Globe className="w-5 h-5 text-gray-400" />
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        {/* Wallet Info Card */}
        <div className="glass rounded-2xl p-6 mb-8">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-sm text-gray-400 mb-1">Total Balance</h2>
              <div className="text-4xl font-bold text-white">{formatValue(totalValue)}</div>
            </div>
            <div className={`flex items-center gap-1 ${change24h >= 0 ? 'text-green-500' : 'text-red-500'}`}>
              {change24h >= 0 ? <ArrowUpRight className="w-5 h-5" /> : <ArrowDownRight className="w-5 h-5" />}
              <span className="font-medium">{change24h >= 0 ? '+' : ''}{change24h.toFixed(2)}%</span>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <span>Address:</span>
            <code className="px-2 py-1 bg-dark-800 rounded font-mono">{formatAddress(address)}</code>
            <button className="text-primary-500 hover:text-primary-400">Copy</button>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          {[
            { icon: ArrowUpRight, label: 'Send', color: 'bg-blue-500' },
            { icon: ArrowDownRight, label: 'Receive', color: 'bg-green-500' },
            { icon: Activity, label: 'Trade', color: 'bg-purple-500' },
            { icon: Zap, label: 'Swap', color: 'bg-orange-500' },
          ].map((action, i) => (
            <button key={i} className="glass rounded-xl p-4 hover:bg-white/10 transition-all flex flex-col items-center gap-2">
              <div className={`w-12 h-12 rounded-full ${action.color} flex items-center justify-center`}>
                <action.icon className="w-6 h-6 text-white" />
              </div>
              <span className="text-sm font-medium">{action.label}</span>
            </button>
          ))}
        </div>

        {/* Supported Blockchains */}
        <div className="glass rounded-2xl p-6 mb-8">
          <h3 className="text-lg font-bold mb-4 flex items-center gap-2">
            <Globe className="w-5 h-5 text-primary-500" />
            Supported Networks
          </h3>
          <div className="flex flex-wrap gap-3">
            {SUPPORTED_BLOCKCHAINS.slice(0, 20).map((chain) => (
              <div 
                key={chain.id}
                className="flex items-center gap-2 px-3 py-2 bg-dark-800 rounded-lg"
              >
                <div 
                  className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  style={{ backgroundColor: `${chain.logoUrl}20`, color: chain.logoUrl }}
                >
                  {chain.symbol.slice(0, 2)}
                </div>
                <span className="text-sm">{chain.name}</span>
              </div>
            ))}
          </div>
          <p className="text-sm text-gray-500 mt-4">
            +{SUPPORTED_BLOCKCHAINS.length - 20} more networks supported
          </p>
        </div>

        {/* Balances */}
        <div className="glass rounded-2xl p-6">
          <h3 className="text-lg font-bold mb-4 flex items-center gap-2">
            <Coins className="w-5 h-5 text-primary-500" />
            Assets
          </h3>
          <div className="space-y-3">
            {balances.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No assets found. Start by receiving or swapping.</p>
            ) : (
              balances.map((balance, i) => (
                <div key={i} className="flex items-center justify-between p-4 bg-dark-800/50 rounded-xl hover:bg-dark-800 transition-colors">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-primary-500/20 flex items-center justify-center text-primary-500 font-bold">
                      {balance.symbol.slice(0, 2)}
                    </div>
                    <div>
                      <div className="font-medium">{balance.symbol}</div>
                      <div className="text-sm text-gray-500">{balance.name}</div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-medium">{balance.balance}</div>
                    <div className="text-sm text-gray-500">{formatValue(balance.value)}</div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
