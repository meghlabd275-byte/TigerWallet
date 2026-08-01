// Swap Page
import React, { useState, useEffect } from 'react';
import { swapApi, walletApi, SwapQuote, Wallet } from '../services/api';
import './SwapPage.css';

const SwapPage: React.FC = () => {
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDT');
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [slippage, setSlippage] = useState('0.5');
  const [tokens, setTokens] = useState<{ symbol: string; name: string; icon: string }[]>([]);
  const [quote, setQuote] = useState<SwapQuote | null>(null);
  const [loading, setLoading] = useState(false);
  const [swapping, setSwapping] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [wallets, setWallets] = useState<Wallet[]>([]);
  const [selectedWallet, setSelectedWallet] = useState<string>('');

  // Load tokens on mount
  useEffect(() => {
    loadTokens();
    loadWallets();
  }, []);

  const loadTokens = async () => {
    try {
      const tokenList = await swapApi.getTokens('ethereum');
      setTokens(tokenList.map(t => ({
        symbol: t.symbol,
        name: t.name,
        icon: getTokenIcon(t.symbol),
      })));
    } catch (err) {
      console.error('Failed to load tokens:', err);
      // Fallback
      setTokens([
        { symbol: 'ETH', name: 'Ethereum', icon: '🔷' },
        { symbol: 'USDT', name: 'Tether USD', icon: '💵' },
        { symbol: 'USDC', name: 'USD Coin', icon: '💲' },
        { symbol: 'BNB', name: 'BNB', icon: '🟡' },
        { symbol: 'SOL', name: 'Solana', icon: '☀️' },
        { symbol: 'MATIC', name: 'Polygon', icon: '🟣' },
      ]);
    }
  };

  const loadWallets = async () => {
    try {
      const walletList = await walletApi.getWallets();
      setWallets(walletList);
      if (walletList.length > 0) {
        setSelectedWallet(walletList[0].id);
      }
    } catch (err) {
      console.error('Failed to load wallets:', err);
    }
  };

  // Get quote when amount or tokens change
  useEffect(() => {
    if (fromAmount && parseFloat(fromAmount) > 0) {
      getQuote();
    }
  }, [fromAmount, fromToken, toToken]);

  const getQuote = async () => {
    setLoading(true);
    try {
      const swapQuote = await swapApi.getQuote(
        fromToken,
        toToken,
        fromAmount,
        parseFloat(slippage)
      );
      setQuote(swapQuote);
      setToAmount(swapQuote.toAmount);
    } catch (err) {
      console.error('Failed to get quote:', err);
      setQuote(null);
    } finally {
      setLoading(false);
    }
  };

  const getTokenIcon = (symbol: string): string => {
    const icons: Record<string, string> = {
      ETH: '🔷', BNB: '🟡', SOL: '☀️', USDT: '💵', USDC: '💲',
      MATIC: '🟣', WBTC: '₿', LINK: '🔗', DOGE: '🐕', XRP: '💜',
    };
    return icons[symbol.toUpperCase()] || '🪙';
  };

  const exchangeTokens = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
    setFromAmount('');
    setToAmount('');
  };

  const handleSwap = async () => {
    if (!fromAmount || !toAmount || !selectedWallet || !quote) {
      setError('Please fill in all fields');
      return;
    }

    setSwapping(true);
    setError(null);

    try {
      const result = await swapApi.execute(
        selectedWallet,
        fromToken,
        toToken,
        fromAmount,
        toAmount,
        quote.route
      );
      alert(`Swap successful! Transaction hash: ${result.hash}`);
      setFromAmount('');
      setToAmount('');
    } catch (err: any) {
      console.error('Swap failed:', err);
      setError(err.message || 'Swap failed. Please try again.');
    } finally {
      setSwapping(false);
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
