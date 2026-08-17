/**
 * Swap Page - DEX Token Swapping
 */

import React, { useState, useEffect } from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService } from '../services/WalletService';

function SwapPage() {
  const { theme } = useTheme();
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDT');
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [slippage, setSlippage] = useState(0.5);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successHash, setSuccessHash] = useState('');
  const [quote, setQuote] = useState<{ priceImpact: number; route: string[] } | null>(null);
  const [password, setPassword] = useState('');

  const tokens = ['ETH', 'USDT', 'USDC', 'MATIC', 'BNB', 'AVAX', 'SOL', 'ARB', 'OP'];
  const chainId = activeWallet?.chain?.chainId ?? 1;

  // Fetch a real quote whenever inputs change.
  useEffect(() => {
    let cancelled = false;
    async function fetchQuote() {
      if (!fromAmount || parseFloat(fromAmount) <= 0) {
        setToAmount('');
        setQuote(null);
        return;
      }
      try {
        const q = await walletService.getSwapQuote(fromToken, toToken, fromAmount, chainId);
        if (cancelled) return;
        setToAmount(q.toAmount);
        setQuote({ priceImpact: q.priceImpact, route: q.route });
        setError(null);
      } catch (err: any) {
        if (cancelled) return;
        setToAmount('');
        setQuote(null);
        setError(err?.response?.data?.error || err?.message || 'Failed to fetch swap quote');
      }
    }
    fetchQuote();
    return () => { cancelled = true; };
  }, [fromToken, toToken, fromAmount, chainId, walletService]);

  const handleSwap = async () => {
    if (!activeWallet) { setError('No active wallet'); return; }
    if (!password) { setError('Wallet password is required to execute the swap'); return; }
    setIsLoading(true);
    setError(null);
    setSuccessHash('');
    try {
      const result = await walletService.swap(
        activeWallet.id,
        fromToken,
        toToken,
        fromAmount,
        password,
        chainId,
      );
      if (!result.txHash) {
        setError('Swap executed but no transaction hash was returned by the backend');
      } else {
        setSuccessHash(result.txHash);
        setPassword('');
        setFromAmount('');
        setToAmount('');
        setQuote(null);
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Swap failed');
    } finally {
      setIsLoading(false);
    }
  };

  const swapTokens = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
  };

  return (
    <div className="p-6 max-w-xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Swap</h1>

      {successHash && (
        <div className={`card mb-6 bg-green-500/20 border-green-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <h3 className="font-semibold text-green-500 mb-2">✓ Transaction submitted to the blockchain network</h3>
          <p className="text-sm opacity-70">Tx Hash:</p>
          <p className="font-mono text-xs break-all">{successHash}</p>
        </div>
      )}

      <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        {/* From Token */}
        <div className="mb-4">
          <label className="label">From</label>
          <div className="flex gap-2">
            <select value={fromToken} onChange={(e) => setFromToken(e.target.value)} className="input w-24">
              {tokens.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
            <input
              type="number"
              value={fromAmount}
              onChange={(e) => setFromAmount(e.target.value)}
              placeholder="0.00"
              className="input flex-1"
            />
          </div>
        </div>

        {/* Swap Button */}
        <div className="flex justify-center -my-2 relative z-10">
          <button onClick={swapTokens} className="p-2 rounded-full bg-amber-500 text-black">
            ⇅
          </button>
        </div>

        {/* To Token */}
        <div className="mb-6">
          <label className="label">To</label>
          <div className="flex gap-2">
            <select value={toToken} onChange={(e) => setToToken(e.target.value)} className="input w-24">
              {tokens.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
            <input
              type="number"
              value={toAmount}
              readOnly
              placeholder="0.00"
              className="input flex-1"
            />
          </div>
        </div>

        {/* Slippage */}
        <div className="mb-6">
          <label className="label">Slippage: {slippage}%</label>
          <input
            type="range"
            min="0.1"
            max="5"
            step="0.1"
            value={slippage}
            onChange={(e) => setSlippage(parseFloat(e.target.value))}
            className="w-full"
          />
        </div>

        {/* Wallet password (required by backend /swap/execute) */}
        <div className="mb-6">
          <label className="label">Wallet Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Wallet password"
            className="input w-full"
          />
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-500/10 text-red-500 text-sm">{error}</div>
        )}

        <button onClick={handleSwap} disabled={isLoading || !fromAmount || !password} className="btn btn-primary w-full">
          {isLoading ? 'Swapping...' : 'Swap'}
        </button>
      </div>

      {/* Exchange Info (real quote data) */}
      <div className={`card mt-4 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Exchange Info</h3>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span>Price Impact</span>
            <span>{quote ? `${quote.priceImpact}%` : '—'}</span>
          </div>
          <div className="flex justify-between">
            <span>Route</span>
            <span>{quote && quote.route.length ? quote.route.join(' → ') : `${fromToken} → ${toToken}`}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default SwapPage;
