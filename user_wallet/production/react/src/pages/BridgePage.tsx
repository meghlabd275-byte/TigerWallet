/**
 * Bridge Page - Cross-chain bridging
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';

function BridgePage() {
  const { theme } = useTheme();
  const [fromChain, setFromChain] = useState('ethereum');
  const [toChain, setToChain] = useState('polygon');
  const [amount, setAmount] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const chains = [
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC' },
    { id: 'arbitrum', name: 'Arbitrum', symbol: 'ARB' },
    { id: 'optimism', name: 'Optimism', symbol: 'OP' },
    { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB' },
  ];

  const bridges = [
    { name: 'Stargate', supported: true, fee: '0.1%' },
    { name: 'LayerZero', supported: true, fee: '0.05%' },
    { name: 'Hop Protocol', supported: true, fee: '0.08%' },
    { name: 'Across', supported: true, fee: '0.03%' },
  ];

  const handleBridge = () => {
    setIsLoading(true);
    setTimeout(() => setIsLoading(false), 3000);
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Bridge</h1>

      <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <div className="mb-4">
          <label className="label">From Chain</label>
          <select value={fromChain} onChange={(e) => setFromChain(e.target.value)} className="input">
            {chains.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </div>

        <div className="flex justify-center -my-2">
          <button onClick={() => { const t = fromChain; setFromChain(toChain); setToChain(t); }} className="p-2 rounded-full bg-amber-500 text-black">⇅</button>
        </div>

        <div className="mb-4">
          <label className="label">To Chain</label>
          <select value={toChain} onChange={(e) => setToChain(e.target.value)} className="input">
            {chains.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </div>

        <div className="mb-6">
          <label className="label">Amount</label>
          <input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0.00" className="input" />
        </div>

        <div className="mb-4 p-3 bg-gray-700 rounded-lg">
          <div className="flex justify-between text-sm"><span>Estimated Time</span><span>~5-10 minutes</span></div>
          <div className="flex justify-between text-sm"><span>Bridge Fee</span><span>~0.1%</span></div>
          <div className="flex justify-between text-sm"><span>You Receive</span><span>{amount || '0'}</span></div>
        </div>

        <button onClick={handleBridge} disabled={isLoading || !amount} className="btn btn-primary w-full">
          {isLoading ? 'Processing...' : 'Bridge'}
        </button>
      </div>

      <div className="mt-6">
        <h3 className="font-semibold mb-3">Supported Bridges</h3>
        <div className="space-y-2">
          {bridges.map((b, i) => (
            <div key={i} className={`flex justify-between items-center p-3 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'} rounded-lg`}>
              <span>{b.name}</span>
              <span className="text-sm opacity-60">Fee: {b.fee}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default BridgePage;
