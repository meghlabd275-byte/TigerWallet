'use client';

import React, { useState } from 'react';

interface SimResult {
  success: boolean;
  gasUsed: string;
  gasPrice: string;
  totalCost: string;
  balanceChange: string;
  error?: string;
}

export default function TxSimulation() {
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [token, setToken] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [simulating, setSimulating] = useState(false);
  const [result, setResult] = useState<SimResult | null>(null);

  const simulateTx = async () => {
    if (!from || !to || !amount) return;
    setSimulating(true);
    setResult(null);
    await new Promise(r => setTimeout(r, 2000));
    
    const isSuccess = Math.random() > 0.1;
    const gasUsed = Math.floor(Math.random() * 50000 + 21000).toString();
    const gasPrice = (Math.random() * 30 + 10).toFixed(2);
    const totalCost = (parseFloat(gasUsed) * parseFloat(gasPrice) / 1e9).toFixed(6);
    
    setResult({
      success: isSuccess,
      gasUsed,
      gasPrice,
      totalCost,
      balanceChange: `-${amount} ${token}`,
      error: isSuccess ? undefined : 'Insufficient gas or contract revert',
    });
    setSimulating(false);
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4"><div className="flex items-center justify-between h-16"><div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Transaction Simulation</h1></div></div></div>
      </header>
      <div className="max-w-3xl mx-auto px-4 py-8">
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4">Simulate Transaction</h2>
          <div className="space-y-4">
            <div><label className="block text-sm text-slate-500 mb-2">From Address</label><input type="text" value={from} onChange={(e) => setFrom(e.target.value)} placeholder="0x..." className="w-full bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3 font-mono text-sm" /></div>
            <div><label className="block text-sm text-slate-500 mb-2">To Address</label><input type="text" value={to} onChange={(e) => setTo(e.target.value)} placeholder="0x..." className="w-full bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3 font-mono text-sm" /></div>
            <div className="grid grid-cols-2 gap-4">
              <div><label className="block text-sm text-slate-500 mb-2">Token</label><select value={token} onChange={(e) => setToken(e.target.value)} className="w-full bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3"><option>ETH</option><option>USDT</option><option>USDC</option><option>BNB</option><option>MATIC</option></select></div>
              <div><label className="block text-sm text-slate-500 mb-2">Amount</label><input type="number" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0.0" className="w-full bg-slate-100 dark:bg-slate-700 rounded-lg px-4 py-3" /></div>
            </div>
            <button onClick={simulateTx} disabled={simulating || !from || !to || !amount} className="w-full bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white font-semibold py-3 rounded-lg">{simulating ? 'Simulating...' : 'Simulate Transaction'}</button>
          </div>
        </div>
        
        {simulating && (
          <div className="bg-white dark:bg-slate-800 rounded-lg p-8 text-center">
            <div className="animate-spin w-12 h-12 border-4 border-orange-500 border-t-transparent rounded-full mx-auto mb-4"></div>
            <p className="text-slate-500">Running simulation...</p>
          </div>
        )}
        
        {result && (
          <div className={`bg-white dark:bg-slate-800 rounded-lg p-6 ${result.success ? 'border-2 border-green-500' : 'border-2 border-red-500'}`}>
            <div className="flex items-center gap-3 mb-4">
              <div className={`text-3xl ${result.success ? '✅' : '❌'}`}></div>
              <div><div className="text-xl font-semibold">{result.success ? 'Transaction Would Succeed' : 'Transaction Would Fail'}</div><div className="text-sm text-slate-500">{result.error}</div></div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3"><div className="text-sm text-slate-500">Gas Used</div><div className="font-semibold">{result.gasUsed}</div></div>
              <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3"><div className="text-sm text-slate-500">Gas Price</div><div className="font-semibold">{result.gasPrice} Gwei</div></div>
              <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3"><div className="text-sm text-slate-500">Total Cost</div><div className="font-semibold">{result.totalCost} ETH</div></div>
              <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3"><div className="text-sm text-slate-500">Balance Change</div><div className="font-semibold text-red-500">{result.balanceChange}</div></div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
