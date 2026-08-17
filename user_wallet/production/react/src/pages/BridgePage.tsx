/**
 * Bridge Page - Cross-chain bridging
 *
 * The canonical wallet-api backend has no bridge HTTP service (go/bridge is
 * a library, not a server). WalletService.bridge()/getBridges() throw honest
 * errors, so this page surfaces those errors instead of fabricating
 * transactions, fees, or bridge lists.
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useWallet } from '../contexts/WalletContext';
import { WalletService } from '../services/WalletService';

function BridgePage() {
  const { theme } = useTheme();
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());
  const [fromChain, setFromChain] = useState('ethereum');
  const [toChain, setToChain] = useState('polygon');
  const [amount, setAmount] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successHash, setSuccessHash] = useState('');

  const chains = [
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC' },
    { id: 'arbitrum', name: 'Arbitrum', symbol: 'ARB' },
    { id: 'optimism', name: 'Optimism', symbol: 'OP' },
    { id: 'avalanche', name: 'Avalanche', symbol: 'AVAX' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB' },
  ];

  const handleBridge = async () => {
    setIsLoading(true);
    setError(null);
    setSuccessHash('');
    try {
      if (!activeWallet) throw new Error('No active wallet selected');
      const result = await walletService.bridge(activeWallet.id, fromChain, toChain, 'native', amount);
      setSuccessHash(result.txHash || result.bridgeTxHash || '');
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Bridge failed');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Bridge</h1>

      {successHash && (
        <div className={`card mb-6 bg-green-500/20 border-green-500 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <h3 className="font-semibold text-green-500 mb-2">✓ Transaction submitted to the blockchain network</h3>
          <p className="text-sm opacity-70">Tx Hash:</p>
          <p className="font-mono text-xs break-all">{successHash}</p>
        </div>
      )}

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
          <div className="flex justify-between text-sm"><span>You Receive</span><span>{amount || '0'}</span></div>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-900/40 text-red-200 rounded-lg text-sm">{error}</div>
        )}

        <button onClick={handleBridge} disabled={isLoading || !amount} className="btn btn-primary w-full">
          {isLoading ? 'Processing...' : 'Bridge'}
        </button>
      </div>
    </div>
  );
}

export default BridgePage;
