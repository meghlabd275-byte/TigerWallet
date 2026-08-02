/**
 * TigerWallet - Bridge Page
 * Complete cross-chain bridge functionality
 */

import React, { useState, useCallback } from 'react';
import { useTheme } from '../stores/ThemeStore';

interface BridgeRoute {
  from: string;
  to: string;
  tokens: string[];
  provider: string;
  time: string;
}

interface BridgeQuote {
  fromChain: string;
  toChain: string;
  token: string;
  sendAmount: string;
  receiveAmount: string;
  bridgeFee: string;
  estimatedTime: string;
  provider: string;
}

const BridgePage: React.FC = () => {
  const { theme } = useTheme();
  
  const [fromChain, setFromChain] = useState('ethereum');
  const [toChain, setToChain] = useState('polygon');
  const [selectedToken, setSelectedToken] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [quote, setQuote] = useState<BridgeQuote | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const chains = [
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC' },
    { id: 'arbitrum', name: 'Arbitrum', symbol: 'ETH' },
    { id: 'optimism', name: 'Optimism', symbol: 'ETH' },
    { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB' },
    { id: 'base', name: 'Base', symbol: 'ETH' },
    { id: 'solana', name: 'Solana', symbol: 'SOL' },
  ];

  const routes: BridgeRoute[] = [
    { from: 'ethereum', to: 'polygon', tokens: ['ETH', 'USDT', 'USDC'], provider: 'Stargate', time: '10-15m' },
    { from: 'ethereum', to: 'arbitrum', tokens: ['ETH', 'USDC'], provider: 'LayerZero', time: '15-20m' },
    { from: 'ethereum', to: 'optimism', tokens: ['ETH', 'USDC'], provider: 'LayerZero', time: '15-20m' },
    { from: 'ethereum', to: 'avalanche', tokens: ['ETH', 'USDC'], provider: 'Axelar', time: '20-30m' },
    { from: 'ethereum', to: 'bsc', tokens: ['ETH', 'BNB'], provider: 'Stargate', time: '5-10m' },
    { from: 'polygon', to: 'ethereum', tokens: ['MATIC', 'USDC'], provider: 'Stargate', time: '10-15m' },
    { from: 'bsc', to: 'ethereum', tokens: ['BNB', 'ETH'], provider: 'Stargate', time: '5-10m' },
    { from: 'arbitrum', to: 'ethereum', tokens: ['ETH'], provider: 'LayerZero', time: '15-20m' },
    { from: 'solana', to: 'ethereum', tokens: ['SOL', 'USDC'], provider: 'Wormhole', time: '15-20m' },
  ];

  const getAvailableTokens = (from: string, to: string): string[] => {
    const route = routes.find(r => r.from === from && r.to === to);
    return route?.tokens || [];
  };

  const getProvider = (from: string, to: string): string => {
    const route = routes.find(r => r.from === from && r.to === to);
    return route?.provider || 'Stargate';
  };

  const getEstimatedTime = (from: string, to: string): string => {
    const route = routes.find(r => r.from === from && r.to === to);
    return route?.time || '15-30m';
  };

  const handleGetQuote = useCallback(async () => {
    if (!amount) return;
    
    setIsLoading(true);
    try {
      // Simulate quote calculation
      const feePercent = 0.001;
      const bridgeFee = parseFloat(amount) * feePercent;
      const receiveAmount = parseFloat(amount) - bridgeFee;

      setQuote({
        fromChain,
        toChain,
        token: selectedToken,
        sendAmount: amount,
        receiveAmount: receiveAmount.toFixed(6),
        bridgeFee: bridgeFee.toFixed(6),
        estimatedTime: getEstimatedTime(fromChain, toChain),
        provider: getProvider(fromChain, toChain)
      });
    } finally {
      setIsLoading(false);
    }
  }, [fromChain, toChain, selectedToken, amount]);

  const handleSwapChains = useCallback(() => {
    setFromChain(toChain);
    setToChain(fromChain);
    setQuote(null);
  }, [fromChain, toChain]);

  const handleBridge = useCallback(async () => {
    if (!quote) return;
    
    setIsLoading(true);
    try {
      // Simulate bridge transaction
      alert(`Bridging ${quote.sendAmount} ${quote.token} from ${fromChain} to ${toChain}`);
      setAmount('');
      setQuote(null);
    } finally {
      setIsLoading(false);
    }
  }, [quote, fromChain, toChain]);

  const availableTokens = getAvailableTokens(fromChain, toChain);

  return (
    <div className={`bridge-page ${theme}`}>
      <div className="page-header">
        <h1>🌉 Bridge</h1>
        <p>Transfer tokens across different blockchains</p>
      </div>

      {/* Bridge Form */}
      <div className="bridge-container">
        {/* From */}
        <div className="chain-section">
          <label>From</label>
          <select
            value={fromChain}
            onChange={(e) => {
              setFromChain(e.target.value);
              setQuote(null);
              const tokens = getAvailableTokens(e.target.value, toChain);
              if (!tokens.includes(selectedToken)) {
                setSelectedToken(tokens[0] || '');
              }
            }}
            className="chain-select"
          >
            {chains.map(chain => (
              <option key={chain.id} value={chain.id}>{chain.name}</option>
            ))}
          </select>
        </div>

        {/* Swap Button */}
        <button className="swap-btn" onClick={handleSwapChains}>
          ⇄
        </button>

        {/* To */}
        <div className="chain-section">
          <label>To</label>
          <select
            value={toChain}
            onChange={(e) => {
              setToChain(e.target.value);
              setQuote(null);
              const tokens = getAvailableTokens(fromChain, e.target.value);
              if (!tokens.includes(selectedToken)) {
                setSelectedToken(tokens[0] || '');
              }
            }}
            className="chain-select"
          >
            {chains.map(chain => (
              <option key={chain.id} value={chain.id}>{chain.name}</option>
            ))}
          </select>
        </div>

        {/* Token & Amount */}
        <div className="token-section">
          <div className="token-row">
            <select
              value={selectedToken}
              onChange={(e) => {
                setSelectedToken(e.target.value);
                setQuote(null);
              }}
              className="token-select"
            >
              {availableTokens.map(token => (
                <option key={token} value={token}>{token}</option>
              ))}
            </select>
            <input
              type="number"
              value={amount}
              onChange={(e) => {
                setAmount(e.target.value);
                setQuote(null);
              }}
              placeholder="0.00"
              className="amount-input"
            />
          </div>
        </div>

        {/* Get Quote Button */}
        <button
          className="btn btn-primary btn-large"
          onClick={handleGetQuote}
          disabled={isLoading || !amount}
        >
          {isLoading ? 'Getting Quote...' : 'Get Quote'}
        </button>

        {/* Quote Display */}
        {quote && (
          <div className="quote-section">
            <h3>Bridge Quote</h3>
            
            <div className="quote-details">
              <div className="quote-row">
                <span>You Send</span>
                <span>{quote.sendAmount} {quote.token}</span>
              </div>
              <div className="quote-row">
                <span>Bridge Fee</span>
                <span>{quote.bridgeFee} {quote.token}</span>
              </div>
              <div className="quote-row highlight">
                <span>You Receive</span>
                <span>{quote.receiveAmount} {quote.token}</span>
              </div>
              <div className="quote-row">
                <span>Estimated Time</span>
                <span>{quote.estimatedTime}</span>
              </div>
              <div className="quote-row">
                <span>Provider</span>
                <span>{quote.provider}</span>
              </div>
            </div>

            <button
              className="btn btn-primary btn-large"
              onClick={handleBridge}
              disabled={isLoading}
            >
              {isLoading ? 'Bridging...' : 'Bridge Now'}
            </button>
          </div>
        )}
      </div>

      {/* Supported Routes */}
      <div className="section">
        <h2>Supported Routes</h2>
        <div className="routes-grid">
          {routes.slice(0, 8).map((route, index) => (
            <div key={index} className="route-card">
              <div className="route-chains">
                <span className="from-chain">{route.from}</span>
                <span className="arrow">→</span>
                <span className="to-chain">{route.to}</span>
              </div>
              <div className="route-tokens">
                {route.tokens.join(', ')}
              </div>
              <div className="route-info">
                <span>{route.provider}</span>
                <span>~{route.time}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Security Notice */}
      <div className="section security-notice">
        <h3>🔒 Security Tips</h3>
        <ul>
          <li>Always verify the recipient address before bridging</li>
          <li>Bridging times vary by network congestion</li>
          <li>Some bridges may require approvals - ensure sufficient gas on destination chain</li>
          <li>Start with a small test amount first</li>
        </ul>
      </div>

      <style>{`
        .bridge-page {
          padding: 20px;
          max-width: 600px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 24px;
        }

        .page-header h1 {
          font-size: 28px;
          margin-bottom: 8px;
        }

        .bridge-container {
          background: var(--card-bg, #1e1e2e);
          border-radius: 16px;
          padding: 24px;
        }

        .chain-section {
          margin-bottom: 16px;
        }

        .chain-section label {
          display: block;
          margin-bottom: 8px;
          font-weight: 600;
          color: var(--text-secondary, #ccc);
        }

        .chain-select, .token-select {
          width: 100%;
          padding: 14px;
          background: var(--input-bg, #2a2a3e);
          border: 1px solid var(--border-color, #333);
          border-radius: 12px;
          color: var(--text-primary, #fff);
          font-size: 16px;
          cursor: pointer;
        }

        .swap-btn {
          display: block;
          margin: 16px auto;
          width: 48px;
          height: 48px;
          border-radius: 50%;
          background: var(--primary-color, #6c5ce7);
          border: none;
          color: white;
          font-size: 24px;
          cursor: pointer;
          transition: transform 0.2s;
        }

        .swap-btn:hover {
          transform: rotate(180deg);
        }

        .token-section {
          margin: 24px 0;
        }

        .token-row {
          display: grid;
          grid-template-columns: 100px 1fr;
          gap: 12px;
        }

        .amount-input {
          padding: 14px;
          background: var(--input-bg, #2a2a3e);
          border: 1px solid var(--border-color, #333);
          border-radius: 12px;
          color: var(--text-primary, #fff);
          font-size: 18px;
          text-align: right;
        }

        .quote-section {
          margin-top: 24px;
          padding: 20px;
          background: var(--info-bg, #2a2a4e);
          border-radius: 12px;
        }

        .quote-section h3 {
          margin-bottom: 16px;
        }

        .quote-details {
          margin-bottom: 20px;
        }

        .quote-row {
          display: flex;
          justify-content: space-between;
          padding: 10px 0;
          border-bottom: 1px solid var(--border-color, #333);
        }

        .quote-row.highlight {
          font-size: 18px;
          font-weight: 700;
          color: var(--primary-color, #6c5ce7);
        }

        .section {
          background: var(--card-bg, #1e1e2e);
          border-radius: 12px;
          padding: 20px;
          margin-top: 20px;
        }

        .section h2 {
          font-size: 20px;
          margin-bottom: 16px;
        }

        .routes-grid {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 12px;
        }

        .route-card {
          background: var(--input-bg, #2a2a3e);
          border-radius: 8px;
          padding: 12px;
        }

        .route-chains {
          display: flex;
          align-items: center;
          gap: 8px;
          font-weight: 600;
          margin-bottom: 6px;
        }

        .route-tokens {
          font-size: 12px;
          color: var(--text-secondary, #ccc);
          margin-bottom: 6px;
        }

        .route-info {
          display: flex;
          justify-content: space-between;
          font-size: 12px;
          color: var(--text-muted, #888);
        }

        .security-notice ul {
          padding-left: 20px;
        }

        .security-notice li {
          margin-bottom: 8px;
          color: var(--text-secondary, #ccc);
        }
      `}</style>
    </div>
  );
};

export default BridgePage;
