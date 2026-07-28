// Home Page - Dashboard
// Complete portfolio overview with light/dark theme

import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import './HomePage.css';

interface Token {
  id: string;
  name: string;
  symbol: string;
  balance: string;
  value: number;
  change24h: number;
  icon: string;
}

interface Transaction {
  id: string;
  type: 'send' | 'receive' | 'swap';
  amount: string;
  token: string;
  time: string;
  status: 'completed' | 'pending';
}

const HomePage: React.FC = () => {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [totalValue, setTotalValue] = useState(0);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = () => {
    setTokens([
      { id: '1', name: 'Ethereum', symbol: 'ETH', balance: '5.5', value: 16500, change24h: 2.5, icon: '🔷' },
      { id: '2', name: 'BNB', symbol: 'BNB', balance: '12.8', value: 3840, change24h: -1.2, icon: '🟡' },
      { id: '3', name: 'Solana', symbol: 'SOL', balance: '85', value: 9350, change24h: 5.8, icon: '☀️' },
      { id: '4', name: 'USDT', symbol: 'USDT', balance: '15000', value: 15000, change24h: 0.01, icon: '💵' },
      { id: '5', name: 'Polygon', symbol: 'MATIC', balance: '5000', value: 4500, change24h: -0.5, icon: '🟣' },
    ]);

    setTransactions([
      { id: '1', type: 'receive', amount: '+2.5 ETH', token: 'ETH', time: '2 min ago', status: 'completed' },
      { id: '2', type: 'send', amount: '-500 USDT', token: 'USDT', time: '1 hour ago', status: 'completed' },
      { id: '3', type: 'swap', amount: '1 ETH → 3000 USDC', token: 'ETH', time: '3 hours ago', status: 'completed' },
      { id: '4', type: 'receive', amount: '+50 SOL', token: 'SOL', time: '5 hours ago', status: 'completed' },
    ]);

    setTotalValue(49190);
  };

  const getChangeColor = (change: number) => {
    return change >= 0 ? 'positive' : 'negative';
  };

  return (
    <div className="home-page">
      <div className="page-header">
        <h1>Dashboard</h1>
      </div>

      {/* Portfolio Card */}
      <div className="portfolio-card">
        <div className="portfolio-header">
          <span className="portfolio-label">Total Balance</span>
          <div className="network-selector">
            <span className="network-icon">🔷</span>
            <span>Ethereum</span>
          </div>
        </div>
        <div className="portfolio-value">
          <span className="currency">$</span>
          <span className="amount">{totalValue.toLocaleString()}</span>
        </div>
        <div className="portfolio-change positive">
          <span>+2.5%</span>
          <span>($1,225)</span>
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
