/**
 * History Page - Transaction history.
 * All data is fetched live from the canonical backend (go/wallet_api :8443)
 * via WalletService. No mock data.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useWallet } from '../contexts/WalletContext';
import { WalletService, Transaction } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

function HistoryPage() {
  const { theme } = useTheme();
  const { activeWallet, wallets } = useWallet();
  const [walletService] = useState(() => new WalletService());
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [filter, setFilter] = useState('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!activeWallet) {
      setTransactions([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const txs = await walletService
        .getTransactions(activeWallet.id, 1, 50)
        .catch(() => [] as Transaction[]);
      setTransactions(txs);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load transactions');
      setTransactions([]);
    } finally {
      setLoading(false);
    }
  }, [activeWallet, walletService]);

  useEffect(() => {
    load();
  }, [load]);

  const txType = (tx: Transaction): string => {
    if (!activeWallet) return 'Swap';
    return tx.from?.toLowerCase() === activeWallet.address?.toLowerCase() ? 'Send' : 'Receive';
  };

  const filtered =
    filter === 'all'
      ? transactions
      : transactions.filter((t) => txType(t).toLowerCase() === filter);

  const fmtTime = (ts: string) => {
    if (!ts) return '';
    const n = Number(ts);
    const d = n > 1e9 ? new Date(n * 1000) : new Date(n);
    if (isNaN(d.getTime())) return ts;
    return d.toLocaleString();
  };

  const fmtValue = (tx: Transaction) => {
    const isSend = txType(tx) === 'Send';
    const sign = isSend ? '-' : '+';
    return `${sign}${tx.value} ${tx.token || ''}`.trim();
  };

  const totalValue = wallets.reduce((sum, w) => sum + (w.balanceUSD || 0), 0);

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Transaction History</h1>

      {error && (
        <div className={`mb-4 p-3 rounded-lg text-sm ${theme === 'dark' ? 'bg-red-900/40 text-red-300' : 'bg-red-100 text-red-700'}`}>
          {error}
        </div>
      )}

      <div className="flex gap-2 mb-6 overflow-x-auto">
        {['all', 'send', 'receive'].map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-4 py-2 rounded-lg whitespace-nowrap ${filter === f ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}
          >
            {f.charAt(0).toUpperCase() + f.slice(1)}
          </button>
        ))}
      </div>

      {loading ? (
        <LoadingSpinner label="Loading transactions..." />
      ) : filtered.length === 0 ? (
        <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'} text-center py-12 opacity-60`}>
          No transactions yet.
        </div>
      ) : (
        <div className="space-y-3">
          {filtered.map((tx) => {
            const type = txType(tx);
            return (
              <div key={tx.id + tx.timestamp} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
                <div className="flex justify-between items-start">
                  <div>
                    <div className="flex items-center gap-2">
                      <span
                        className={`px-2 py-1 rounded text-xs ${
                          type === 'Send'
                            ? 'bg-red-500/20 text-red-500'
                            : type === 'Receive'
                            ? 'bg-green-500/20 text-green-500'
                            : 'bg-blue-500/20 text-blue-500'
                        }`}
                      >
                        {type}
                      </span>
                      <span className="font-mono text-sm opacity-60">
                        {tx.hash ? `${tx.hash.slice(0, 10)}...` : tx.id}
                      </span>
                    </div>
                    <p className="text-sm opacity-60 mt-1">{fmtTime(tx.timestamp)}</p>
                    {tx.to && <p className="text-xs opacity-40 mt-1 font-mono">to: {tx.to}</p>}
                  </div>
                  <div className="text-right">
                    <p className={`font-bold ${type === 'Receive' ? 'text-green-500' : ''}`}>{fmtValue(tx)}</p>
                    <span className={`badge ${tx.status === 'confirmed' ? 'badge-success' : 'badge-warning'}`}>
                      {tx.status}
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {totalValue > 0 && (
        <p className="text-xs opacity-40 mt-6 text-center">
          Showing {filtered.length} transaction{filtered.length !== 1 ? 's' : ''}
        </p>
      )}
    </div>
  );
}

export default HistoryPage;
