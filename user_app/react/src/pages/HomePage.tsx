// Home Page - Dashboard
// Complete portfolio overview with light/dark theme
// PRODUCTION-READY - Real blockchain data integration

import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { WalletService, TransactionService, SwapService, wsService } from '../services/walletService';
import { walletApi, TokenBalance, Transaction } from '../services/api';
import './HomePage.css';

interface Token {
  id: string;
  name: string;
  symbol: string;
  balance: string;
  value: number;
  change24h: number;
  icon: string;
  address: string;
  chainId: number;
  decimals: number;
  priceUSD: number;
}

interface Transaction {
  id: string;
  type: 'send' | 'receive' | 'swap';
  amount: string;
  token: string;
  time: string;
  status: 'completed' | 'pending';
  hash: string;
}

const CHAIN_ICONS: Record<string, string> = {
  ethereum: '🔷',
  bsc: '🟡',
  polygon: '🟣',
  arbitrum: '🔵',
  optimism: '⬆️',
  avalanche: '🔺',
  solana: '☀️',
  base: '🟦',
  fantom: '👻',
  ton: '📱',
  tron: '🔴',
  bitcoin: '₿',
};

const HomePage: React.FC = () => {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [totalValue, setTotalValue] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedChain, setSelectedChain] = useState('ethereum');
  const [wallets, setWallets] = useState<any[]>([]);
  const [priceUpdates, setPriceUpdates] = useState<Record<string, number>>({});

  // Real-time price update handler
  const handlePriceUpdate = useCallback((data: { symbol: string; price: number }) => {
    setPriceUpdates(prev => ({ ...prev, [data.symbol]: data.price }));
  }, []);

  useEffect(() => {
    loadData();
    
    // Subscribe to real-time price updates
    wsService.subscribe('price_update', handlePriceUpdate);
    
    // Subscribe to new transactions
    wsService.subscribe('new_transaction', (tx: any) => {
      setTransactions(prev => [formatTransaction(tx), ...prev.slice(0, 3)]);
    });

    return () => {
      wsService.unsubscribe('price_update', handlePriceUpdate);
    };
  }, [handlePriceUpdate]);

  const formatTransaction = (tx: any): Transaction => {
    const now = Date.now();
    const txTime = new Date(tx.timestamp * 1000);
    const diffMs = now - txTime.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    let time: string;
    if (diffMins < 1) time = 'Just now';
    else if (diffMins < 60) time = `${diffMins} min ago`;
    else if (diffHours < 24) time = `${diffHours} hours ago`;
    else time = `${diffDays} days ago`;

    const type = tx.from === wallets[0]?.address ? 'send' : 'receive';
    const amount = type === 'send' 
      ? `-${tx.amount} ${tx.symbol}`
      : `+${tx.amount} ${tx.symbol}`;

    return {
      id: tx.id || tx.hash,
      type: tx.type || type,
      amount,
      token: tx.symbol,
      time,
      status: tx.status === 'confirmed' ? 'completed' : 'pending',
      hash: tx.hash,
    };
  };

  const loadData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      // Get all wallets
      const allWallets = await WalletService.getWallets();
      setWallets(allWallets);
      
      if (allWallets.length === 0) {
        setLoading(false);
        return;
      }

      // Get wallet details for selected chain
      const chainWallets = allWallets.filter((w: any) => w.chain === selectedChain);
      const primaryWallet = chainWallets[0] || allWallets[0];
      
      if (!primaryWallet) {
        setLoading(false);
        return;
      }

      // Get wallet details with balance
      const walletDetails = await WalletService.getWalletDetails(primaryWallet.id);
      
      // Get real token balances with live prices
      const tokenBalances: Token[] = (walletDetails.tokens || []).map((token: TokenBalance) => {
        const price = priceUpdates[token.symbol] || token.priceUSD || token.balanceUSD / parseFloat(token.balance);
        const value = parseFloat(token.balance) * price;
        return {
          id: token.address || token.symbol,
          name: token.name,
          symbol: token.symbol,
          balance: token.balance,
          value: value || token.balanceUSD,
          change24h: token.priceChange24h || 0,
          icon: CHAIN_ICONS[selectedChain] || '💰',
          address: token.address,
          chainId: token.chainId,
          decimals: token.decimals,
          priceUSD: price,
        };
      });

      // Calculate total portfolio value across all chains
      let total = 0;
      for (const wallet of allWallets) {
        const details = await WalletService.getWalletDetails(wallet.id).catch(() => null);
        if (details?.tokens) {
          for (const token of details.tokens) {
            const price = priceUpdates[token.symbol] || token.priceUSD;
            total += parseFloat(token.balance) * (price || 0);
          }
        }
        total += parseFloat(wallet.balance || '0') * (priceUpdates[wallet.symbol] || 0);
      }

      // Get transaction history
      const txHistory = await TransactionService.getHistory(primaryWallet.id, 1, 10);
      const formattedTransactions: Transaction[] = txHistory.slice(0, 4).map(formatTransaction);

      setTokens(tokenBalances);
      setTransactions(formattedTransactions);
      setTotalValue(total || walletDetails.balanceUSD || 0);
      
    } catch (err: any) {
      console.error('Failed to load wallet data:', err);
      setError(err.message || 'Failed to load wallet data');
      
      // Fallback: Try to get cached data from localStorage
      const cachedWallets = localStorage.getItem('cached_wallets');
      if (cachedWallets) {
        const parsed = JSON.parse(cachedWallets);
        setWallets(parsed.wallets || []);
        setTokens(parsed.tokens || []);
        setTransactions(parsed.transactions || []);
        setTotalValue(parsed.totalValue || 0);
      }
    } finally {
      setLoading(false);
    }
  };

  const refreshData = async () => {
    await loadData();
  };

  const getChangeColor = (change: number) => {
    return change >= 0 ? 'positive' : 'negative';
  };

  // Calculate 24h change based on token prices
  const calculate24hChange = () => {
    if (tokens.length === 0) return 0;
    let totalChange = 0;
    let totalWeight = 0;
    for (const token of tokens) {
      totalChange += (token.change24h || 0) * (token.value || 0);
      totalWeight += token.value || 0;
    }
    return totalWeight > 0 ? totalChange / totalWeight : 0;
  };

  const change24h = calculate24hChange();
  const changeValue = totalValue * (change24h / 100);

  if (loading) {
    return (
      <div className="home-page">
        <div className="page-header">
          <h1>Dashboard</h1>
        </div>
        <div className="loading-container">
          <div className="loading-spinner"></div>
          <p>Loading your portfolio...</p>
        </div>
      </div>
    );
  }

  if (error && wallets.length === 0) {
    return (
      <div className="home-page">
        <div className="page-header">
          <h1>Dashboard</h1>
        </div>
        <div className="error-container">
          <p>⚠️ {error}</p>
          <p>Please create or import a wallet to get started.</p>
          <div className="quick-actions">
            <Link to="/wallet" className="action-button">
              <div className="action-icon">👛</div>
              <span>Create Wallet</span>
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="home-page">
      <div className="page-header">
        <h1>Dashboard</h1>
        <button className="refresh-button" onClick={refreshData} disabled={loading}>
          {loading ? '↻' : '🔄'}
        </button>
      </div>

      {/* Portfolio Card */}
      <div className="portfolio-card">
        <div className="portfolio-header">
          <span className="portfolio-label">Total Balance</span>
          <div className="network-selector">
            <select 
              value={selectedChain} 
              onChange={(e) => setSelectedChain(e.target.value)}
              className="chain-select"
            >
              {wallets.map((w: any) => (
                <option key={w.id} value={w.chain}>
                  {CHAIN_ICONS[w.chain] || '💰'} {w.chain}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="portfolio-value">
          <span className="currency">$</span>
          <span className="amount">{totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
        </div>
        <div className={`portfolio-change ${change24h >= 0 ? 'positive' : 'negative'}`}>
          <span>{change24h >= 0 ? '+' : ''}{change24h.toFixed(2)}%</span>
          <span>({change24h >= 0 ? '+$' : '-$'}{Math.abs(changeValue).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })})</span>
          <span>24h</span>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="quick-actions">
        <Link to="/send" className="action-button">
          <div className="action-icon">📤</div>
          <span>Send</span>
        </Link>
        <Link to="/receive" className="action-button">
          <div className="action-icon">📥</div>
          <span>Receive</span>
        </Link>
        <Link to="/swap" className="action-button">
          <div className="action-icon">🔄</div>
          <span>Swap</span>
        </Link>
        <Link to="/dapps" className="action-button">
          <div className="action-icon">🌐</div>
          <span>DApps</span>
        </Link>
      </div>

      {/* Assets */}
      <div className="section">
        <div className="section-header">
          <h2>Assets</h2>
          <Link to="/wallet" className="see-all">See All</Link>
        </div>
        <div className="assets-list">
          {tokens.map(token => (
            <div key={token.id} className="asset-item">
              <div className="asset-icon">{token.icon}</div>
              <div className="asset-info">
                <span className="asset-name">{token.name}</span>
                <span className="asset-balance">{token.balance} {token.symbol}</span>
              </div>
              <div className="asset-value">
                <span className="value-usd">${token.value.toLocaleString()}</span>
                <span className={`value-change ${getChangeColor(token.change24h)}`}>
                  {token.change24h >= 0 ? '+' : ''}{token.change24h}%
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Recent Transactions */}
      <div className="section">
        <div className="section-header">
          <h2>Recent Transactions</h2>
        </div>
        <div className="transactions-list">
          {transactions.map(tx => (
            <div key={tx.id} className="transaction-item">
              <div className={`tx-icon ${tx.type}`}>
                {tx.type === 'send' ? '📤' : tx.type === 'receive' ? '📥' : '🔄'}
              </div>
              <div className="tx-info">
                <span className="tx-type">
                  {tx.type === 'send' ? 'Sent' : tx.type === 'receive' ? 'Received' : 'Swapped'}
                </span>
                <span className="tx-time">{tx.time}</span>
              </div>
              <div className={`tx-amount ${tx.type}`}>
                {tx.amount}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default HomePage;
