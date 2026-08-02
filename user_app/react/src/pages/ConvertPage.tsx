// Convert Page - Token Conversion (Like Binance Convert)
// Simple one-click token conversion

import React, { useState, useEffect } from 'react';
import './ConvertPage.css';

interface Token {
  symbol: string;
  name: string;
  balance: number;
  icon: string;
  usdValue?: number;
}

interface ConversionRate {
  from: string;
  to: string;
  rate: number;
  inverseRate: number;
  fee: number;
}

interface ConversionHistory {
  id: string;
  fromToken: string;
  toToken: string;
  fromAmount: number;
  toAmount: number;
  rate: number;
  timestamp: number;
  status: 'completed' | 'pending' | 'failed';
}

const AVAILABLE_TOKENS: Token[] = [
  { symbol: 'USDT', name: 'Tether USD', balance: 50000, icon: '💵' },
  { symbol: 'USDC', name: 'USD Coin', balance: 25000, icon: '💳' },
  { symbol: 'BTC', name: 'Bitcoin', balance: 0.5, icon: '₿' },
  { symbol: 'ETH', name: 'Ethereum', balance: 5, icon: 'Ξ' },
  { symbol: 'BNB', name: 'BNB', balance: 50, icon: '⬡' },
  { symbol: 'SOL', name: 'Solana', balance: 100, icon: '◎' },
  { symbol: 'XRP', name: 'Ripple', balance: 10000, icon: '✕' },
  { symbol: 'DOGE', name: 'Dogecoin', balance: 100000, icon: 'Ð' },
  { symbol: 'ADA', name: 'Cardano', balance: 5000, icon: '₳' },
  { symbol: 'AVAX', name: 'Avalanche', balance: 500, icon: '🔺' },
  { symbol: 'DOT', name: 'Polkadot', balance: 1000, icon: '●' },
  { symbol: 'LINK', name: 'Chainlink', balance: 500, icon: '⬡' },
  { symbol: 'MATIC', name: 'Polygon', balance: 5000, icon: '⬡' },
  { symbol: 'LTC', name: 'Litecoin', balance: 100, icon: 'Ł' },
  { symbol: 'UNI', name: 'Uniswap', balance: 1000, icon: '🦄' },
  { symbol: 'ATOM', name: 'Cosmos', balance: 500, icon: '⚛' },
  { symbol: 'XLM', name: 'Stellar', balance: 5000, icon: '✦' },
  { symbol: 'NEAR', name: 'NEAR Protocol', balance: 500, icon: 'Ⓝ' },
];

// Simulated prices (in USD)
const PRICES: {[key: string]: number} = {
  USDT: 1, USDC: 1, BTC: 43250, ETH: 2280, BNB: 312,
  SOL: 98.75, XRP: 0.62, DOGE: 0.082, ADA: 0.58, AVAX: 38.20,
  DOT: 7.85, LINK: 14.50, MATIC: 0.92, LTC: 72.30, UNI: 6.25,
  ATOM: 10.45, XLM: 0.125, NEAR: 3.25,
};

const ConvertPage: React.FC = () => {
  const [tokens, setTokens] = useState<Token[]>(AVAILABLE_TOKENS);
  const [fromToken, setFromToken] = useState<Token>(AVAILABLE_TOKENS[0]);
  const [toToken, setToToken] = useState<Token>(AVAILABLE_TOKENS[1]);
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [rate, setRate] = useState<ConversionRate | null>(null);
  const [showFromSelector, setShowFromSelector] = useState(false);
  const [showToSelector, setShowToSelector] = useState(false);
  const [converting, setConverting] = useState(false);
  const [conversionSuccess, setConversionSuccess] = useState<ConversionHistory | null>(null);
  const [history, setHistory] = useState<ConversionHistory[]>([]);
  const [activeTab, setActiveTab] = useState<'convert' | 'history'>('convert');

  // Calculate USD values
  useEffect(() => {
    const tokensWithValues = tokens.map(t => ({
      ...t,
      usdValue: t.balance * (PRICES[t.symbol] || 0)
    }));
    setTokens(tokensWithValues);
  }, []);

  // Calculate conversion rate
  useEffect(() => {
    if (!fromAmount) {
      setToAmount('');
      setRate(null);
      return;
    }

    const fromPrice = PRICES[fromToken.symbol] || 0;
    const toPrice = PRICES[toToken.symbol] || 0;
    
    if (fromPrice && toPrice) {
      const conversionRate = fromPrice / toPrice;
      const fee = parseFloat(fromAmount) * 0.001; // 0.1% fee
      const netAmount = parseFloat(fromAmount) - fee;
      const receivedAmount = netAmount * conversionRate;
      
      setToAmount(receivedAmount.toFixed(toToken.symbol === 'BTC' || toToken.symbol === 'ETH' ? 6 : 4));
      setRate({
        from: fromToken.symbol,
        to: toToken.symbol,
        rate: conversionRate,
        inverseRate: 1 / conversionRate,
        fee
      });
    }
  }, [fromAmount, fromToken, toToken]);

  const handleSwap = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
    setFromAmount('');
    setToAmount('');
  };

  const handleConvert = async () => {
    if (!fromAmount || !toAmount) return;
    
    setConverting(true);
    
    // Simulate conversion
    await new Promise(resolve => setTimeout(resolve, 2000));
    
    // Update balances
    const newHistory: ConversionHistory = {
      id: Date.now().toString(),
      fromToken: fromToken.symbol,
      toToken: toToken.symbol,
      fromAmount: parseFloat(fromAmount),
      toAmount: parseFloat(toAmount),
      rate: rate?.rate || 1,
      timestamp: Date.now(),
      status: 'completed'
    };
    
    setHistory([newHistory, ...history]);
    setConversionSuccess(newHistory);
    setFromAmount('');
    setToAmount('');
    setConverting(false);
  };

  const handleSelectMax = () => {
    setFromAmount(fromToken.balance.toString());
  };

  const getFilteredTokens = (exclude?: string) => {
    return tokens.filter(t => t.symbol !== exclude);
  };

  return (
    <div className="convert-page">
      <div className="convert-header">
        <h1>🔄 Convert</h1>
        <p>Instantly convert your tokens at the best rate</p>
      </div>

      <div className="tabs">
        <button 
          className={activeTab === 'convert' ? 'active' : ''} 
          onClick={() => setActiveTab('convert')}
        >
          Convert
        </button>
        <button 
          className={activeTab === 'history' ? 'active' : ''} 
          onClick={() => setActiveTab('history')}
        >
          History
        </button>
      </div>

      {activeTab === 'convert' && (
        <div className="convert-section">
          <div className="convert-card">
            <div className="convert-input-group">
              <div className="input-header">
                <span className="input-label">You Pay</span>
                <span className="balance">
                  Balance: {fromToken.balance.toFixed(4)} {fromToken.symbol}
                </span>
              </div>
              
              <div className="token-selector" onClick={() => setShowFromSelector(true)}>
                <span className="token-icon">{fromToken.icon}</span>
                <span className="token-symbol">{fromToken.symbol}</span>
                <span className="dropdown">▼</span>
              </div>
              
              <div className="amount-input">
                <input
                  type="number"
                  value={fromAmount}
                  onChange={(e) => setFromAmount(e.target.value)}
                  placeholder="0.00"
                />
                <button className="max-btn" onClick={handleSelectMax}>
                  MAX
                </button>
              </div>
              
              {fromAmount && (
                <div className="usd-value">
                  ~${(parseFloat(fromAmount) * (PRICES[fromToken.symbol] || 0)).toFixed(2)} USD
                </div>
              )}
            </div>

            <div className="swap-button" onClick={handleSwap}>
              <span>⇅</span>
            </div>

            <div className="convert-input-group">
              <div className="input-header">
                <span className="input-label">You Receive</span>
                <span className="balance">
                  Balance: {toToken.balance.toFixed(4)} {toToken.symbol}
                </span>
              </div>
              
              <div className="token-selector" onClick={() => setShowToSelector(true)}>
                <span className="token-icon">{toToken.icon}</span>
                <span className="token-symbol">{toToken.symbol}</span>
                <span className="dropdown">▼</span>
              </div>
              
              <div className="amount-input">
                <input
                  type="number"
                  value={toAmount}
                  readOnly
                  placeholder="0.00"
                  className="received-amount"
                />
              </div>
              
              {toAmount && (
                <div className="usd-value">
                  ~${(parseFloat(toAmount) * (PRICES[toToken.symbol] || 0)).toFixed(2)} USD
                </div>
              )}
            </div>

            {rate && fromAmount && (
              <div className="rate-info">
                <div className="rate-row">
                  <span>Rate</span>
                  <span>1 {fromToken.symbol} = {rate.rate.toFixed(6)} {toToken.symbol}</span>
                </div>
                <div className="rate-row">
                  <span>Fee (0.1%)</span>
                  <span>{rate.fee.toFixed(4)} {fromToken.symbol}</span>
                </div>
                <div className="rate-row">
                  <span>Slippage</span>
                  <span>&lt; 0.5%</span>
                </div>
              </div>
            )}

            <button 
              className="convert-btn"
              onClick={handleConvert}
              disabled={!fromAmount || !toAmount || converting || parseFloat(fromAmount) > fromToken.balance}
            >
              {converting ? 'Converting...' : `Convert to ${toToken.symbol}`}
            </button>
          </div>

          <div className="info-card">
            <h3>Why Use Convert?</h3>
            <ul>
              <li>⚡ Instant conversion - no waiting for order matching</li>
              <li>💱 Best market rates automatically</li>
              <li>🔒 Secure - powered by TigerWallet</li>
              <li>💰 Low fees - only 0.1% conversion fee</li>
            </ul>
          </div>
        </div>
      )}

      {activeTab === 'history' && (
        <div className="history-section">
          {history.length === 0 ? (
            <div className="empty-history">
              <span className="empty-icon">📋</span>
              <h3>No Conversion History</h3>
              <p>Your token conversions will appear here</p>
            </div>
          ) : (
            <div className="history-list">
              {history.map(item => (
                <div key={item.id} className="history-item">
                  <div className="history-icon">🔄</div>
                  <div className="history-details">
                    <div className="conversion-pair">
                      {item.fromToken} → {item.toToken}
                    </div>
                    <div className="conversion-amount">
                      {item.fromAmount} {item.fromToken} → {item.toAmount.toFixed(6)} {item.toToken}
                    </div>
                    <div className="conversion-time">
                      {new Date(item.timestamp).toLocaleString()}
                    </div>
                  </div>
                  <div className={`status ${item.status}`}>
                    {item.status}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Token Selector Modal - From */}
      {showFromSelector && (
        <div className="token-selector-modal" onClick={() => setShowFromSelector(false)}>
          <div className="selector-content" onClick={e => e.stopPropagation()}>
            <h3>Select Token</h3>
            <div className="token-list">
              {getFilteredTokens(toToken.symbol).map(token => (
                <div 
                  key={token.symbol} 
                  className="token-option"
                  onClick={() => {
                    setFromToken(token);
                    setShowFromSelector(false);
                  }}
                >
                  <span className="token-icon">{token.icon}</span>
                  <div className="token-info">
                    <span className="token-symbol">{token.symbol}</span>
                    <span className="token-name">{token.name}</span>
                  </div>
                  <span className="token-balance">{token.balance.toFixed(4)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Token Selector Modal - To */}
      {showToSelector && (
        <div className="token-selector-modal" onClick={() => setShowToSelector(false)}>
          <div className="selector-content" onClick={e => e.stopPropagation()}>
            <h3>Select Token</h3>
            <div className="token-list">
              {getFilteredTokens(fromToken.symbol).map(token => (
                <div 
                  key={token.symbol} 
                  className="token-option"
                  onClick={() => {
                    setToToken(token);
                    setShowToSelector(false);
                  }}
                >
                  <span className="token-icon">{token.icon}</span>
                  <div className="token-info">
                    <span className="token-symbol">{token.symbol}</span>
                    <span className="token-name">{token.name}</span>
                  </div>
                  <span className="token-balance">{token.balance.toFixed(4)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Success Modal */}
      {conversionSuccess && (
        <div className="success-modal" onClick={() => setConversionSuccess(null)}>
          <div className="success-content" onClick={e => e.stopPropagation()}>
            <div className="success-icon">✅</div>
            <h2>Conversion Complete!</h2>
            <div className="conversion-summary">
              <span>{conversionSuccess.fromAmount} {conversionSuccess.fromToken}</span>
              <span>→</span>
              <span>{conversionSuccess.toAmount.toFixed(6)} {conversionSuccess.toToken}</span>
            </div>
            <p>Your tokens have been converted successfully</p>
            <button className="done-btn" onClick={() => setConversionSuccess(null)}>
              Done
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default ConvertPage;
