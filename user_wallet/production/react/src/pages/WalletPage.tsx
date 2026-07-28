/**
 * Wallet Page - Token Balances & Management
 */

import React, { useState, useEffect } from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';
import { Token, Chain } from '../services/WalletService';

function WalletPage() {
  const { activeWallet, refreshBalances, isLoading } = useWallet();
  const { theme, toggleTheme } = useTheme();
  const [tokens, setTokens] = useState<Token[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    if (activeWallet?.tokens) {
      setTokens(activeWallet.tokens);
    }
  }, [activeWallet]);

  const filteredTokens = tokens.filter(token =>
    token.symbol.toLowerCase().includes(searchQuery.toLowerCase()) ||
    token.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const totalBalance = tokens.reduce((acc, token) => acc + token.balanceUSD, 0);

  return (
    <div className="p-6">
      {/* Balance Card */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold">Total Balance</h2>
          <button onClick={toggleTheme} className="p-2 rounded-lg bg-amber-500 text-black">
            {theme === 'dark' ? '☀️' : '🌙'}
          </button>
        </div>
        <div className="text-4xl font-bold text-amber-500">
          ${totalBalance.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
        </div>
        <div className="flex gap-2 mt-4">
          <button className="btn btn-primary flex-1">Send</button>
          <button className="btn btn-secondary flex-1">Receive</button>
          <button className="btn btn-secondary flex-1">Swap</button>
        </div>
      </div>

      {/* Chain Selector */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Select Network</h3>
        <div className="flex flex-wrap gap-2">
          {['Ethereum', 'Polygon', 'BNB Chain', 'Arbitrum', 'Solana', 'Avalanche'].map(chain => (
            <button key={chain} className={`px-4 py-2 rounded-lg text-sm ${
              activeWallet?.chain.name === chain 
                ? 'bg-amber-500 text-black' 
                : theme === 'dark' ? 'bg-slate-700' : 'bg-gray-200'
            }`}>
              {chain}
            </button>
          ))}
        </div>
      </div>

      {/* Search */}
      <div className="mb-4">
        <input
          type="text"
          placeholder="Search tokens..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className={`input w-full ${theme === 'dark' ? 'bg-slate-800 border-slate-700' : 'bg-white'}`}
        />
      </div>

      {/* Token List */}
      <div className="space-y-3">
        {filteredTokens.map(token => (
          <div key={token.address} className={`token-item ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <div className="token-info">
              <div className="token-icon bg-amber-500 rounded-full w-10 h-10 flex items-center justify-center text-black font-bold">
                {token.symbol.slice(0, 2)}
              </div>
              <div>
                <div className="font-semibold">{token.name}</div>
                <div className="text-sm opacity-60">{token.symbol}</div>
              </div>
            </div>
            <div className="token-balance text-right">
              <div className="font-semibold">{parseFloat(token.balance).toFixed(6)}</div>
              <div className="text-sm opacity-60">${token.balanceUSD.toFixed(2)}</div>
            </div>
          </div>
        ))}
      </div>

      {isLoading && (
        <div className="flex justify-center py-4">
          <div className="spinner"></div>
        </div>
      )}
    </div>
  );
}

export default WalletPage;
