// Convert Page - Token Conversion (Like Binance Convert)
// Simple one-click token conversion - Connected to backend convert service

import React, { useState, useEffect, useCallback } from 'react';
import './ConvertPage.css';
import { api } from '../services/api';

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

// Token icons mapping
const TOKEN_ICONS: {[key: string]: string} = {
  USDT: '💵', USDC: '💳', BTC: '₿', ETH: 'Ξ', BNB: '⬡',
  SOL: '◎', XRP: '✕', DOGE: 'Ð', ADA: '₳', AVAX: '🔺',
  DOT: '●', LINK: '⬡', MATIC: '⬡', LTC: 'Ł', UNI: '🦄',
  ATOM: '⚛', XLM: '✦', NEAR: 'Ⓝ', TRX: '⏳', BTT: '🔹'
};

const ConvertPage: React.FC = () => {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [fromToken, setFromToken] = useState<Token | null>(null);
  const [toToken, setToToken] = useState<Token | null>(null);
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [rate, setRate] = useState<ConversionRate | null>(null);
  const [showFromSelector, setShowFromSelector] = useState(false);
  const [showToSelector, setShowToSelector] = useState(false);
  const [converting, setConverting] = useState(false);
  const [conversionSuccess, setConversionSuccess] = useState<ConversionHistory | null>(null);
  const [history, setHistory] = useState<ConversionHistory[]>([]);
  const [activeTab, setActiveTab] = useState<'convert' | 'history'>('convert');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load tokens from backend
  const loadTokens = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('http://localhost:8443/api/v1/convert/pairs');
      const data = await response.json();
      
      if (data.pairs && Array.isArray(data.pairs)) {
        const uniqueTokens = new Map<string, Token>();
        
        data.pairs.forEach((pair: any) => {
          if (!uniqueTokens.has(pair.from)) {
            uniqueTokens.set(pair.from, {
              symbol: pair.from,
              name: pair.from,
              balance: 0,
              icon: TOKEN_ICONS[pair.from] || '🪙'
            });
          }
          if (!uniqueTokens.has(pair.to)) {
            uniqueTokens.set(pair.to, {
              symbol: pair.to,
              name: pair.to,
              balance: 0,
              icon: TOKEN_ICONS[pair.to] || '🪙'
            });
          }
        });
        
        const tokenList = Array.from(uniqueTokens.values());
        setTokens(tokenList);
        
        if (tokenList.length > 0 && !fromToken) {
          const usdt = tokenList.find(t => t.symbol === 'USDT');
          const usdc = tokenList.find(t => t.symbol === 'USDC');
          setFromToken(usdt || tokenList[0]);
          setToToken(usdc || tokenList[1] || tokenList[0]);
        }
      }
    } catch (err) {
      console.error('Failed to load convert pairs:', err);
      setError('Unable to load conversion pairs. Please ensure the backend service is running.');
    } finally {
      setLoading(false);
    }
  }, [fromToken]);

  // Load user balance
  const loadBalance = useCallback(async () => {
    if (!fromToken) return;
    try {
      const token = localStorage.getItem('user_token');
      if (!token) return;
      
      const response = await fetch('http://localhost:8443/api/v1/wallets/balance', {
        headers: { Authorization: `Bearer ${token}` }
      });
      const data = await response.json();
      
      if (data.tokens) {
        setTokens(prev => prev.map(t => {
          const tokenData = data.tokens.find((x: any) => x.symbol === t.symbol);
          return tokenData ? { ...t, balance: parseFloat(tokenData.balance) || 0 } : t;
        }));
      }
    } catch (err) {
      console.error('Failed to load balance:', err);
    }
  }, [fromToken]);

  useEffect(() => {
    loadTokens();
  }, [loadTokens]);

  useEffect(() => {
    if (fromToken) {
      loadBalance();
    }
  }, [fromToken, loadBalance]);

  // Calculate USD values using rates from API
  useEffect(() => {
    const fetchPrices = async () => {
      try {
        const response = await fetch('http://localhost:8443/api/v1/prices');
        const data = await response.json();
        if (data.prices) {
          const tokensWithValues = tokens.map(t => ({
            ...t,
            usdValue: t.balance * (data.prices[t.symbol] || 0)
          }));
          setTokens(tokensWithValues);
        }
      } catch (err) {
        console.error('Failed to fetch prices:', err);
      }
    };
    if (tokens.length > 0) {
      fetchPrices();
    }
  }, [tokens]);

  // Fetch conversion rate from API
  useEffect(() => {
    if (!fromAmount || !fromToken || !toToken) {
      setToAmount('');
      setRate(null);
      return;
    }

    const fetchRate = async () => {
      try {
        const response = await fetch(
          `http://localhost:8443/api/v1/convert/rate?from=${fromToken.symbol}&to=${toToken.symbol}&amount=${fromAmount}`
        );
        const data = await response.json();
        
        if (data.rate) {
          const receivedAmount = parseFloat(fromAmount) * data.rate;
          setToAmount(receivedAmount.toFixed(toToken.symbol === 'BTC' || toToken.symbol === 'ETH' ? 6 : 4));
          setRate({
            from: fromToken.symbol,
            to: toToken.symbol,
            rate: data.rate,
            inverseRate: 1 / data.rate,
            fee: data.fee || 0
          });
        }
      } catch (err) {
        console.error('Failed to fetch conversion rate:', err);
      }
    };

    const debounce = setTimeout(fetchRate, 300);
    return () => clearTimeout(debounce);
  }, [fromAmount, fromToken, toToken]);

  const handleSwap = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
    setFromAmount('');
    setToAmount('');
  };

  const handleConvert = async () => {
    if (!fromAmount || !toAmount || !fromToken || !toToken) return;
    
    setConverting(true);
    setError(null);
    
    try {
      const token = localStorage.getItem('user_token');
      const response = await fetch('http://localhost:8443/api/v1/convert', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        },
        body: JSON.stringify({
          fromToken: fromToken.symbol,
          toToken: toToken.symbol,
          fromAmount: parseFloat(fromAmount)
        })
      });
      
      const data = await response.json();
      
      if (data.order) {
        const newHistory: ConversionHistory = {
          id: data.order.id,
          fromToken: fromToken.symbol,
          toToken: toToken.symbol,
          fromAmount: parseFloat(fromAmount),
          toAmount: data.order.toAmount,
          rate: data.order.rate,
          timestamp: Date.now(),
          status: data.order.status
        };
        
        setHistory([newHistory, ...history]);
        setConversionSuccess(newHistory);
        setFromAmount('');
        setToAmount('');
        
        // Reload balance after conversion
        loadBalance();
      }
    } catch (err) {
      console.error('Conversion failed:', err);
      setError('Conversion failed. Please try again.');
    } finally {
      setConverting(false);
    }
  };

  const handleSelectMax = () => {
    if (fromToken) {
      setFromAmount(fromToken.balance.toString());
    }
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

      {loading ? (
        <div className="loading-container">
          <div className="loading">Loading conversion pairs...</div>
        </div>
      ) : error ? (
        <div className="error-container">
          <div className="error-message">{error}</div>
          <button className="btn btn-primary" onClick={loadTokens}>Retry</button>
        </div>
      ) : activeTab === 'convert' && (
        <div className="convert-section">
          <div className="convert-card">
            <div className="convert-input-group">
              <div className="input-header">
                <span className="input-label">You Pay</span>
                <span className="balance">
                  Balance: {fromToken ? fromToken.balance.toFixed(4) : '0.0000'} {fromToken ? fromToken.symbol : ''}
                </span>
              </div>
              
              <div className="token-selector" onClick={() => setShowFromSelector(true)}>
                <span className="token-icon">{fromToken?.icon || '🪙'}</span>
                <span className="token-symbol">{fromToken?.symbol || 'Select'}</span>
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
              
              {fromAmount && fromToken && (
                <div className="usd-value">
                  ~${fromToken.usdValue ? (parseFloat(fromAmount) * fromToken.usdValue).toFixed(2) : '0.00'} USD
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
                  Balance: {toToken ? toToken.balance.toFixed(4) : '0.0000'} {toToken ? toToken.symbol : ''}
                </span>
              </div>
              
              <div className="token-selector" onClick={() => setShowToSelector(true)}>
                <span className="token-icon">{toToken?.icon || '🪙'}</span>
                <span className="token-symbol">{toToken?.symbol || 'Select'}</span>
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
