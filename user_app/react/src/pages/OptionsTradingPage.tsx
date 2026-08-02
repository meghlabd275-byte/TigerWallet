// Options Trading Page - Call/Put Options Trading
// Supports 50,000+ trading pairs

import React, { useState, useEffect, useCallback } from 'react';
import './OptionsTradingPage.css';

interface OptionPair {
  id: string;
  symbol: string;
  base: string;
  quote: string;
  currentPrice: number;
  isPreInstalled: boolean;
}

interface OptionContract {
  id: string;
  symbol: string;
  type: 'call' | 'put';
  strike: number;
  expiry: string;
  expiryLabel: string;
  bid: number;
  ask: number;
  last: number;
  change24h: number;
  volume24h: number;
  openInterest: number;
  impliedVolatility: number;
  delta: number;
  gamma: number;
  theta: number;
}

interface MyOption {
  id: string;
  symbol: string;
  type: 'call' | 'put';
  strike: number;
  expiry: string;
  size: number;
  entryPrice: number;
  currentPrice: number;
  pnl: number;
  pnlPercent: number;
}

const TOP_PAIRS: OptionPair[] = [
  { id: '1', symbol: 'BTC/USDT', base: 'BTC', quote: 'USDT', currentPrice: 43250, isPreInstalled: true },
  { id: '2', symbol: 'ETH/USDT', base: 'ETH', quote: 'USDT', currentPrice: 2280, isPreInstalled: true },
  { id: '3', symbol: 'BNB/USDT', base: 'BNB', quote: 'USDT', currentPrice: 312.50, isPreInstalled: true },
  { id: '4', symbol: 'SOL/USDT', base: 'SOL', quote: 'USDT', currentPrice: 98.75, isPreInstalled: true },
  { id: '5', symbol: 'XRP/USDT', base: 'XRP', quote: 'USDT', currentPrice: 0.62, isPreInstalled: true },
  { id: '6', symbol: 'DOGE/USDT', base: 'DOGE', quote: 'USDT', currentPrice: 0.082, isPreInstalled: true },
  { id: '7', symbol: 'ADA/USDT', base: 'ADA', quote: 'USDT', currentPrice: 0.58, isPreInstalled: true },
  { id: '8', symbol: 'AVAX/USDT', base: 'AVAX', quote: 'USDT', currentPrice: 38.20, isPreInstalled: true },
  { id: '9', symbol: 'DOT/USDT', base: 'DOT', quote: 'USDT', currentPrice: 7.85, isPreInstalled: true },
  { id: '10', symbol: 'LINK/USDT', base: 'LINK', quote: 'USDT', currentPrice: 14.50, isPreInstalled: true },
  { id: '11', symbol: 'MATIC/USDT', base: 'MATIC', quote: 'USDT', currentPrice: 0.92, isPreInstalled: true },
  { id: '12', symbol: 'LTC/USDT', base: 'LTC', quote: 'USDT', currentPrice: 72.30, isPreInstalled: true },
  { id: '13', symbol: 'UNI/USDT', base: 'UNI', quote: 'USDT', currentPrice: 6.25, isPreInstalled: true },
  { id: '14', symbol: 'ATOM/USDT', base: 'ATOM', quote: 'USDT', currentPrice: 10.45, isPreInstalled: true },
  { id: '15', symbol: 'XLM/USDT', base: 'XLM', quote: 'USDT', currentPrice: 0.125, isPreInstalled: true },
  { id: '16', symbol: 'NEAR/USDT', base: 'NEAR', quote: 'USDT', currentPrice: 3.25, isPreInstalled: true },
  { id: '17', symbol: 'APT/USDT', base: 'APT', quote: 'USDT', currentPrice: 9.80, isPreInstalled: true },
  { id: '18', symbol: 'ARB/USDT', base: 'ARB', quote: 'USDT', currentPrice: 1.12, isPreInstalled: true },
  { id: '19', symbol: 'OP/USDT', base: 'OP', quote: 'USDT', currentPrice: 2.45, isPreInstalled: true },
  { id: '20', symbol: 'INJ/USDT', base: 'INJ', quote: 'USDT', currentPrice: 35.50, isPreInstalled: true },
];

// Backend API URL for options trading
const API_BASE_URL = 'https://api.tigerwallet.com/v1/options';

const EXPIRIES = [
  { value: '1h', label: '1 Hour' },
  { value: '4h', label: '4 Hours' },
  { value: '1d', label: '1 Day' },
  { value: '1w', label: '1 Week' },
  { value: '2w', label: '2 Weeks' },
  { value: '1m', label: '1 Month' },
  { value: '3m', label: '3 Months' },
];

const OptionsTradingPage: React.FC = () => {
  const [pairs, setPairs] = useState<OptionPair[]>([]);
  const [selectedPair, setSelectedPair] = useState<OptionPair | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [expiry, setExpiry] = useState('1d');
  const [optionChain, setOptionChain] = useState<OptionContract[]>([]);
  const [showAllPairs, setShowAllPairs] = useState(false);
  const [activeTab, setActiveTab] = useState<'trade' | 'positions' | 'my-options'>('trade');
  const [orderSide, setOrderSide] = useState<'call' | 'put'>('call');
  const [orderSize, setOrderSize] = useState('');
  const [selectedStrike, setSelectedStrike] = useState<number | null>(null);
  const [myOptions, setMyOptions] = useState<MyOption[]>([]);
  const [walletBalance, setWalletBalance] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load trading pairs from backend
  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      setError(null);
      try {
        const token = localStorage.getItem('user_token');
        
        // Load pairs
        const pairsRes = await fetch(`${API_BASE_URL}/pairs`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        
        if (pairsRes.ok) {
          const data = await pairsRes.json();
          if (data.pairs) {
            setPairs(data.pairs);
            if (data.pairs.length > 0 && !selectedPair) {
              setSelectedPair(data.pairs[0]);
            }
          }
        }
        
        // Load my options
        const optionsRes = await fetch(`${API_BASE_URL}/positions`, {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (optionsRes.ok) {
          const optionsData = await optionsRes.json();
          setMyOptions(optionsData.positions || []);
        }
        
        // Load balance
        const balRes = await fetch('https://api.tigerwallet.com/v1/wallets/balance', {
          headers: token ? { Authorization: `Bearer ${token}` } : {}
        });
        if (balRes.ok) {
          const balData = await balRes.json();
          setWalletBalance(balData.balances?.USDT || 0);
        }
      } catch (err) {
        console.error('Failed to load data:', err);
        setError('Unable to connect to options service. Using offline mode.');
        setPairs(TOP_PAIRS);
        if (!selectedPair) setSelectedPair(TOP_PAIRS[0]);
      } finally {
        setLoading(false);
      }
    };
    
    loadData();
  }, []);

  // Generate option chain based on selected pair and expiry
  useEffect(() => {
    const expiryData = EXPIRIES.find(e => e.value === expiry);
    const strikes = generateStrikes(selectedPair.currentPrice);
    const options: OptionContract[] = [];
    
    strikes.forEach(strike => {
      // Call option
      const callPrice = Math.max(0.01, (selectedPair.currentPrice - strike) * 0.5 + Math.random() * 5);
      options.push({
        id: `call-${strike}-${expiry}`,
        symbol: selectedPair.symbol,
        type: 'call',
        strike,
        expiry,
        expiryLabel: expiryData?.label || expiry,
        bid: callPrice * 0.95,
        ask: callPrice * 1.05,
        last: callPrice,
        change24h: (Math.random() - 0.5) * 20,
        volume24h: Math.floor(Math.random() * 1000000),
        openInterest: Math.floor(Math.random() * 500000),
        impliedVolatility: 20 + Math.random() * 60,
        delta: selectedPair.currentPrice > strike ? 0.3 + Math.random() * 0.4 : Math.random() * 0.3,
        gamma: Math.random() * 0.1,
        theta: -Math.random() * 0.5,
      });
      
      // Put option
      const putPrice = Math.max(0.01, (strike - selectedPair.currentPrice) * 0.5 + Math.random() * 5);
      options.push({
        id: `put-${strike}-${expiry}`,
        symbol: selectedPair.symbol,
        type: 'put',
        strike,
        expiry,
        expiryLabel: expiryData?.label || expiry,
        bid: putPrice * 0.95,
        ask: putPrice * 1.05,
        last: putPrice,
        change24h: (Math.random() - 0.5) * 20,
        volume24h: Math.floor(Math.random() * 1000000),
        openInterest: Math.floor(Math.random() * 500000),
        impliedVolatility: 20 + Math.random() * 60,
        delta: selectedPair.currentPrice < strike ? -0.3 - Math.random() * 0.4 : -Math.random() * 0.3,
        gamma: Math.random() * 0.1,
        theta: -Math.random() * 0.5,
      });
    });
    
    setOptionChain(options);
  }, [selectedPair, expiry]);

  const generateStrikes = (price: number): number[] => {
    const strikes: number[] = [];
    const step = price > 1000 ? 500 : price > 100 ? 50 : price > 10 ? 5 : price > 1 ? 0.5 : 0.05;
    const range = price * 0.15;
    
    for (let s = price - range; s <= price + range; s += step) {
      strikes.push(Math.round(s / step) * step);
    }
    return strikes;
  };

  const filteredPairs = pairs.filter(pair => 
    pair.symbol.toLowerCase().includes(searchTerm.toLowerCase()) || 
    pair.base.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const displayPairs = showAllPairs ? filteredPairs : filteredPairs.filter(p => p.isPreInstalled);

  const callOptions = optionChain.filter(o => o.type === 'call').sort((a, b) => a.strike - b.strike);
  const putOptions = optionChain.filter(o => o.type === 'put').sort((a, b) => b.strike - a.strike);

  const handleTrade = (option: OptionContract, side: 'buy' | 'sell') => {
    const newOption: MyOption = {
      id: Date.now().toString(),
      symbol: option.symbol,
      type: option.type,
      strike: option.strike,
      expiry: option.expiryLabel,
      size: parseFloat(orderSize) || 1,
      entryPrice: option.last,
      currentPrice: option.last,
      pnl: 0,
      pnlPercent: 0,
    };
    setMyOptions([...myOptions, newOption]);
    setActiveTab('my-options');
  };

  const selectedOption = optionChain.find(o => o.strike === selectedStrike && o.type === orderSide);

  return (
    <div className="options-trading-page">
      <div className="options-layout">
        {/* Left Panel - Pairs */}
        <div className="pairs-panel">
          <div className="pairs-header">
            <h3>Assets</h3>
            <input
              type="text"
              placeholder="Search pairs..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>
          
          <div className="pairs-list">
            {displayPairs.map(pair => (
              <div
                key={pair.id}
                className={`pair-item ${selectedPair.id === pair.id ? 'active' : ''} ${pair.isPreInstalled ? 'pre-installed' : ''}`}
                onClick={() => setSelectedPair(pair)}
              >
                <div className="pair-info">
                  <span className="pair-symbol">
                    {pair.base}
                    <span className="quote">/{pair.quote}</span>
                  </span>
                  {pair.isPreInstalled && <span className="pre-installed-badge">★</span>}
                </div>
                <div className="pair-price">
                  <span className="price">${pair.currentPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: pair.currentPrice < 1 ? 6 : 2 })}</span>
                </div>
              </div>
            ))}
          </div>
          
          {filteredPairs.length > (showAllPairs ? filteredPairs.length : 20) && (
            <button 
              className="show-more-btn"
              onClick={() => setShowAllPairs(!showAllPairs)}
            >
              {showAllPairs ? 'Show Top Pairs' : `Show All (${filteredPairs.length}+)`}
            </button>
          )}
        </div>

        {/* Center Panel - Option Chain */}
        <div className="option-chain-panel">
          <div className="option-chain-header">
            <div className="current-asset">
              <h2>{selectedPair.symbol}</h2>
              <span className="current-price">${selectedPair.currentPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: selectedPair.currentPrice < 1 ? 6 : 2 })}</span>
            </div>
            
            <div className="expiry-selector">
              {EXPIRIES.map(exp => (
                <button
                  key={exp.value}
                  className={expiry === exp.value ? 'active' : ''}
                  onClick={() => setExpiry(exp.value)}
                >
                  {exp.label}
                </button>
              ))}
            </div>
          </div>

          <div className="option-chain-container">
            <div className="option-chain">
              <div className="chain-header">
                <span>Strike</span>
                <span>Bid</span>
                <span>Ask</span>
                <span>Last</span>
                <span>24h %</span>
                <span>Vol</span>
                <span>OI</span>
                <span>IV</span>
              </div>
              
              <div className="chain-content">
                <div className="calls-section">
                  <h4>Calls</h4>
                  <div className="options-list">
                    {callOptions.map(option => (
                      <div 
                        key={option.id} 
                        className={`option-row call ${selectedStrike === option.strike ? 'selected' : ''}`}
                        onClick={() => setSelectedStrike(option.strike)}
                      >
                        <span className="strike">{option.strike.toLocaleString()}</span>
                        <span className="bid">{option.bid.toFixed(2)}</span>
                        <span className="ask">{option.ask.toFixed(2)}</span>
                        <span className="last">{option.last.toFixed(2)}</span>
                        <span className={`change ${option.change24h >= 0 ? 'positive' : 'negative'}`}>
                          {option.change24h >= 0 ? '+' : ''}{option.change24h.toFixed(1)}%
                        </span>
                        <span className="volume">{(option.volume24h / 1000).toFixed(1)}K</span>
                        <span className="oi">{(option.openInterest / 1000).toFixed(1)}K</span>
                        <span className="iv">{option.impliedVolatility.toFixed(1)}%</span>
                      </div>
                    ))}
                  </div>
                </div>
                
                <div className="underlying-price">
                  <span>${selectedPair.currentPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: selectedPair.currentPrice < 1 ? 6 : 2 })}</span>
                  <span>Current Price</span>
                </div>
                
                <div className="puts-section">
                  <h4>Puts</h4>
                  <div className="options-list">
                    {putOptions.map(option => (
                      <div 
                        key={option.id} 
                        className={`option-row put ${selectedStrike === option.strike ? 'selected' : ''}`}
                        onClick={() => setSelectedStrike(option.strike)}
                      >
                        <span className="strike">{option.strike.toLocaleString()}</span>
                        <span className="bid">{option.bid.toFixed(2)}</span>
                        <span className="ask">{option.ask.toFixed(2)}</span>
                        <span className="last">{option.last.toFixed(2)}</span>
                        <span className={`change ${option.change24h >= 0 ? 'positive' : 'negative'}`}>
                          {option.change24h >= 0 ? '+' : ''}{option.change24h.toFixed(1)}%
                        </span>
                        <span className="volume">{(option.volume24h / 1000).toFixed(1)}K</span>
                        <span className="oi">{(option.openInterest / 1000).toFixed(1)}K</span>
                        <span className="iv">{option.impliedVolatility.toFixed(1)}%</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Right Panel - Order Form */}
        <div className="order-panel">
          <div className="tabs">
            <button 
              className={activeTab === 'trade' ? 'active' : ''} 
              onClick={() => setActiveTab('trade')}
            >
              Trade
            </button>
            <button 
              className={activeTab === 'my-options' ? 'active' : ''} 
              onClick={() => setActiveTab('my-options')}
            >
              My Options ({myOptions.length})
            </button>
          </div>

          {activeTab === 'trade' && (
            <div className="trade-form">
              <div className="balance-info">
                <span>Available: ${walletBalance.toLocaleString()}</span>
              </div>

              <div className="option-type-selector">
                <button
                  className={orderSide === 'call' ? 'call active' : 'call'}
                  onClick={() => setOrderSide('call')}
                >
                  Call
                </button>
                <button
                  className={orderSide === 'put' ? 'put active' : 'put'}
                  onClick={() => setOrderSide('put')}
                >
                  Put
                </button>
              </div>

              {selectedOption && (
                <div className="selected-option-info">
                  <div className="info-row">
                    <span>Type</span>
                    <span className={selectedOption.type}>{selectedOption.type.toUpperCase()}</span>
                  </div>
                  <div className="info-row">
                    <span>Strike</span>
                    <span>${selectedOption.strike.toLocaleString()}</span>
                  </div>
                  <div className="info-row">
                    <span>Expiry</span>
                    <span>{selectedOption.expiryLabel}</span>
                  </div>
                  <div className="info-row">
                    <span>Mark Price</span>
                    <span>${selectedOption.last.toFixed(2)}</span>
                  </div>
                  <div className="info-row">
                    <span>Bid/Ask</span>
                    <span>${selectedOption.bid.toFixed(2)} / ${selectedOption.ask.toFixed(2)}</span>
                  </div>
                  <div className="info-row">
                    <span>Delta</span>
                    <span>{selectedOption.delta.toFixed(3)}</span>
                  </div>
                  <div className="info-row">
                    <span>Theta</span>
                    <span>{selectedOption.theta.toFixed(3)}</span>
                  </div>
                </div>
              )}

              <div className="form-group">
                <label>Contracts</label>
                <input
                  type="number"
                  value={orderSize}
                  onChange={(e) => setOrderSize(e.target.value)}
                  placeholder="Enter number of contracts"
                  min="1"
                />
              </div>

              {selectedOption && (
                <div className="order-summary">
                  <div className="summary-row">
                    <span>Contract Value:</span>
                    <span>${(parseFloat(orderSize) || 0) * selectedOption.last * selectedPair.currentPrice * 0.01}</span>
                  </div>
                  <div className="summary-row">
                    <span>Premium:</span>
                    <span>${(parseFloat(orderSize) || 0) * selectedOption.ask}</span>
                  </div>
                </div>
              )}

              <div className="order-buttons">
                <button 
                  className="buy-btn"
                  disabled={!selectedOption}
                  onClick={() => selectedOption && handleTrade(selectedOption, 'buy')}
                >
                  Buy Call
                </button>
                <button 
                  className="sell-btn"
                  disabled={!selectedOption}
                  onClick={() => selectedOption && handleTrade(selectedOption, 'sell')}
                >
                  Sell Call
                </button>
              </div>
            </div>
          )}

          {activeTab === 'my-options' && (
            <div className="my-options-list">
              {myOptions.length === 0 ? (
                <div className="empty-state">
                  <p>No options positions</p>
                  <p>Trade options to see your positions here</p>
                </div>
              ) : (
                myOptions.map(option => (
                  <div key={option.id} className="option-position">
                    <div className="position-header">
                      <span className={`type ${option.type}`}>{option.type.toUpperCase()}</span>
                      <span className="symbol">{option.symbol}</span>
                      <span className="expiry">{option.expiry}</span>
                    </div>
                    <div className="position-details">
                      <div className="detail">
                        <span>Strike</span>
                        <span>${option.strike.toLocaleString()}</span>
                      </div>
                      <div className="detail">
                        <span>Size</span>
                        <span>{option.size}</span>
                      </div>
                      <div className="detail">
                        <span>Entry</span>
                        <span>${option.entryPrice.toFixed(2)}</span>
                      </div>
                      <div className="detail">
                        <span>PnL</span>
                        <span className={option.pnl >= 0 ? 'positive' : 'negative'}>
                          ${option.pnl.toFixed(2)}
                        </span>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default OptionsTradingPage;
