// Wallet Page - Complete Token Management
import React, { useState, useEffect } from 'react';
import './WalletPage.css';

interface Token {
  id: string;
  name: string;
  symbol: string;
  balance: string;
  value: number;
  change24h: number;
  icon: string;
  contract?: string;
}

const WalletPage: React.FC = () => {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [walletAddress, setWalletAddress] = useState('');

  useEffect(() => {
    loadTokens();
    generateAddress();
  }, []);

  const loadTokens = () => {
    setTokens([
      { id: '1', name: 'Ethereum', symbol: 'ETH', balance: '5.5', value: 16500, change24h: 2.5, icon: '🔷' },
      { id: '2', name: 'BNB', symbol: 'BNB', balance: '12.8', value: 3840, change24h: -1.2, icon: '🟡' },
      { id: '3', name: 'Solana', symbol: 'SOL', balance: '85', value: 9350, change24h: 5.8, icon: '☀️' },
      { id: '4', name: 'Tether USD', symbol: 'USDT', balance: '15000', value: 15000, change24h: 0.01, icon: '💵', contract: '0xdAC17F958D2ee523a2206206994597C13D831ec7' },
      { id: '5', name: 'USD Coin', symbol: 'USDC', balance: '8000', value: 8000, change24h: 0, icon: '💲', contract: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48' },
      { id: '6', name: 'Polygon', symbol: 'MATIC', balance: '5000', value: 4500, change24h: -0.5, icon: '🟣' },
      { id: '7', name: 'Wrapped Bitcoin', symbol: 'WBTC', balance: '0.5', value: 21500, change24h: 1.8, icon: '₿' },
      { id: '8', name: 'Chainlink', symbol: 'LINK', balance: '500', value: 7500, change24h: 3.2, icon: '🔗' },
    ]);
  };

  const generateAddress = () => {
    setWalletAddress('0x742d35Cc6634C0532925a3b844Bc9e7595f1234');
  };

  const filteredTokens = tokens.filter(token =>
    token.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    token.symbol.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const totalValue = tokens.reduce((acc, token) => acc + token.value, 0);

  const copyAddress = () => {
    navigator.clipboard.writeText(walletAddress);
    alert('Address copied!');
  };

  return (
    <div className="wallet-page">
      <div className="page-header">
        <h1>Wallet</h1>
      </div>

      {/* Wallet Address Card */}
      <div className="address-card">
        <div className="address-header">
          <span className="network-badge">🔷 Ethereum</span>
          <button className="copy-btn" onClick={copyAddress}>📋 Copy</button>
        </div>
        <div className="address-value">{walletAddress}</div>
        <div className="total-balance">
          <span className="label">Total Balance</span>
          <span className="value">${totalValue.toLocaleString()}</span>
        </div>
      </div>

      {/* Search */}
      <div className="search-bar">
        <span>🔍</span>
        <input
          type="text"
          placeholder="Search tokens..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {/* Token List */}
      <div className="tokens-section">
        <h2>Assets ({filteredTokens.length})</h2>
        <div className="tokens-list">
          {filteredTokens.map(token => (
            <div key={token.id} className="token-item">
              <div className="token-icon">{token.icon}</div>
              <div className="token-info">
                <span className="token-name">{token.name}</span>
                <span className="token-balance">{token.balance} {token.symbol}</span>
              </div>
              <div className="token-value">
                <span className="value-usd">${token.value.toLocaleString()}</span>
                <span className={`value-change ${token.change24h >= 0 ? 'positive' : 'negative'}`}>
                  {token.change24h >= 0 ? '+' : ''}{token.change24h}%
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default WalletPage;
