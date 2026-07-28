/**
 * Swap Page - DEX Token Swapping
 */

import React, { useState } from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';

function SwapPage() {
  const { theme } = useTheme();
  const [fromToken, setFromToken] = useState('ETH');
  const [toToken, setToToken] = useState('USDT');
  const [fromAmount, setFromAmount] = useState('');
  const [toAmount, setToAmount] = useState('');
  const [slippage, setSlippage] = useState(0.5);
  const [isLoading, setIsLoading] = useState(false);

  const tokens = ['ETH', 'USDT', 'USDC', 'MATIC', 'BNB', 'AVAX', 'SOL', 'ARB', 'OP'];

  const handleSwap = async () => {
    setIsLoading(true);
    // Real swap implementation would call backend
    setTimeout(() => {
      setIsLoading(false);
    }, 2000);
  };

  const swapTokens = () => {
    const temp = fromToken;
    setFromToken(toToken);
    setToToken(temp);
  };

  return (
    <div className="p-6 max-w-xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Swap</h1>

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

        <button onClick={handleSwap} disabled={isLoading || !fromAmount} className="btn btn-primary w-full">
          {isLoading ? 'Swapping...' : 'Swap'}
        </button>
      </div>

      {/* Exchange Info */}
      <div className={`card mt-4 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Exchange Info</h3>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between"><span>Price Impact</span><span>0.05%</span></div>
          <div className="flex justify-between"><span>LP Fee</span><span>0.3%</span></div>
          <div className="flex justify-between"><span>Route</span><span>{fromToken} → {toToken}</span></div>
        </div>
      </div>
    </div>
  );
}

export default SwapPage;
