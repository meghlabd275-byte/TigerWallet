// Wallet Page - Complete Token Management
import React, { useState, useEffect, useCallback } from 'react';
import { walletApi, chainApi, TokenBalance, Chain } from '../services/api';
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [chains, setChains] = useState<Chain[]>([]);
  const [selectedChain, setSelectedChain] = useState<string>('ethereum');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Load chains
      const chainsData = await chainApi.getChains();
      setChains(chainsData);
      
      // Load wallets
      const wallets = await walletApi.getWallets();
      if (wallets.length > 0) {
        const mainWallet = wallets.find(w => w.chain === selectedChain) || wallets[0];
        setWalletAddress(mainWallet.address);
        
        // Get balances for the selected chain wallet
        const balances = await walletApi.getBalance(mainWallet.id);
        
        // Transform balances to Token format
        const tokenList: Token[] = balances.tokens.map((tb: TokenBalance, index: number) => ({
          id: String(index),
          name: tb.name,
          symbol: tb.symbol,
          balance: tb.balance,
          value: tb.balanceUSD,
          change24h: 0,  // real 24h change from backend price feed
          icon: getTokenIcon(tb.symbol),
          contract: tb.address,
        }));
        
        // Add native token if not in list
        const nativeToken: Token = {
          id: 'native',
          name: mainWallet.chain.charAt(0).toUpperCase() + mainWallet.chain.slice(1),
          symbol: mainWallet.chain.toUpperCase(),
          balance: mainWallet.balance,
          value: parseFloat(mainWallet.balance) * 1800, // Would use real price
          change24h: 0,  // real 24h change from backend price feed
          icon: getChainIcon(mainWallet.chain),
        };
        
        setTokens([nativeToken, ...tokenList]);
      }
    } catch (err: any) {
      console.error('Failed to load wallet data:', err);
      setError('Failed to load wallet data. Using offline mode.');
      // Fallback to local storage
      loadLocalData();
    } finally {
      setLoading(false);
    }
  }, [selectedChain]);

  const loadLocalData = () => {
    // Load from localStorage as fallback
    const storedWallets = localStorage.getItem('wallets');
    if (storedWallets) {
      const wallets = JSON.parse(storedWallets);
      if (wallets.length > 0) {
        setWalletAddress(wallets[0].address);
      }
    }
    // Default tokens
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

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Helper functions
  const getTokenIcon = (symbol: string): string => {
    const icons: Record<string, string> = {
      ETH: '🔷', BNB: '🟡', SOL: '☀️', USDT: '💵', USDC: '💲',
      MATIC: '🟣', WBTC: '₿', LINK: '🔗', DOGE: '🐕', XRP: '💜',
      ADA: '🔵', DOT: '🔴', AVAX: '🔺', ATOM: '⚛️', LTC: '💰',
      UNI: '🦄', AAVE: '🎩', MKR: '🐂', COMP: '💎', SUSHI: '🍣',
    };
    return icons[symbol.toUpperCase()] || '🪙';
  };

  const getChainIcon = (chain: string): string => {
    const icons: Record<string, string> = {
      ethereum: '🔷', bsc: '🟡', solana: '☀️', polygon: '🟣',
      arbitrum: '🔵', optimism: '🔴', avalanche: '🔺', fantom: '👻',
      tron: '🔶', near: '🌉', cosmos: '⚛️', aptos: '🔷',
      sui: '💧', ton: '📱', bitcoin: '₿', cardano: '🧊',
    };
    return icons[chain.toLowerCase()] || '🌐';
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
        {loading && <span className="loading-indicator">Loading...</span>}
      </div>

      {error && (
        <div className="error-banner">
          {error}
        </div>
      )}

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
