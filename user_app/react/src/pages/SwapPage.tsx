// Swap Page
import React, { useState } from 'react';
import './SwapPage.css';

const SwapPage: React.FC = () => {
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDT');
  const [fromAmount, setFromAmount] = useState('');
  const [slippage, setSlippage] = useState('0.5');

  const tokens = [
    { symbol: 'ETH', name: 'Ethereum', icon: '🔷' },
    { symbol: 'USDT', name: 'Tether USD', icon: '💵' },
    { symbol: 'USDC', name: 'USD Coin', icon: '💲' },
    { symbol: 'BNB', name: 'BNB', icon: '🟡' },
    { symbol: 'SOL', name: 'Solana', icon: '☀️' },
    { symbol: 'MATIC', name: 'Polygon', icon: '🟣' },
  ];

  const exchangeTokens = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
  };

  const handleSwap = () => {
    if (fromAmount) {
      alert(`Swapping ${fromAmount} ${fromToken} to ${toToken}`);
    }
  };

  return (
    <div className="swap-page">
      <div className="page-header">
        <h1>Swap</h1>
      </div>

      <div className="swap-form">
        {/* From Token */}
        <div className="swap-section from-section">
          <div className="section-header">
            <label>You Pay</label>
            <span className="balance">Balance: 5.5</span>
          </div>
          <div className="token-input">
            <input
              type="number"
              placeholder="0.0"
              value={fromAmount}
              onChange={(e) => setFromAmount(e.target.value)}
              className="amount-input"
            />
            <select
              value={fromToken}
              onChange={(e) => setFromToken(e.target.value)}
              className="token-select"
            >
              {tokens.map(token => (
                <option key={token.symbol} value={token.symbol}>
                  {token.icon} {token.symbol}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Exchange Button */}
        <button className="exchange-btn" onClick={exchangeTokens}>
          ⇅
        </button>

        {/* To Token */}
        <div className="swap-section to-section">
          <div className="section-header">
            <label>You Receive</label>
          </div>
          <div className="token-input">
            <input
              type="number"
              placeholder="0.0"
              className="amount-input"
              value={fromAmount ? (parseFloat(fromAmount) * 3000).toFixed(2) : ''}
              readOnly
            />
            <select
              value={toToken}
              onChange={(e) => setToToken(e.target.value)}
              className="token-select"
            >
              {tokens.map(token => (
                <option key={token.symbol} value={token.symbol}>
                  {token.icon} {token.symbol}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Rate Info */}
        <div className="rate-info">
          <span>1 {fromToken} ≈ 3000 {toToken}</span>
        </div>

        {/* Slippage */}
        <div className="slippage-section">
          <label>Slippage Tolerance</label>
          <div className="slippage-options">
            <button
              className={slippage === '0.5' ? 'selected' : ''}
              onClick={() => setSlippage('0.5')}
            >0.5%</button>
            <button
              className={slippage === '1' ? 'selected' : ''}
              onClick={() => setSlippage('1')}
            >1%</button>
            <button
              className={slippage === '3' ? 'selected' : ''}
              onClick={() => setSlippage('3')}
            >3%</button>
          </div>
        </div>

        {/* Swap Button */}
        <button className="btn btn-primary btn-full" onClick={handleSwap}>
          Swap {fromToken} → {toToken}
        </button>
      </div>
    </div>
  );
};

export default SwapPage;
