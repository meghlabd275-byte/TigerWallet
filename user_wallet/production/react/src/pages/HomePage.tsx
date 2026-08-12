/**
 * HomePage - wallet dashboard.
 * Shows total portfolio value, per-wallet balances, recent activity, quick
 * actions. All data is fetched live from the canonical backend (go/wallet_api
 * :8443) via WalletService -- no mock data, no stubs.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useWallet } from '../contexts/WalletContext';
import { WalletService, Transaction } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

const HomePage: React.FC = () => {
  const navigate = useNavigate();
  const { activeWallet, wallets, refreshBalances } = useWallet();
  const [walletService] = useState(() => new WalletService());
  const [recentTxs, setRecentTxs] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!activeWallet) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [txs] = await Promise.all([
        walletService.getTransactions(activeWallet.id, 1, 5).catch(() => [] as Transaction[]),
      ]);
      setRecentTxs(txs);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load activity');
    } finally {
      setLoading(false);
    }
  }, [activeWallet, walletService]);

  useEffect(() => {
    load();
  }, [load]);

  const totalValue = wallets.reduce((sum, w) => sum + (w.balanceUSD || 0), 0);

  const quickActions = [
    { label: 'Send', path: '/send', icon: 'M12 19l9 2-9-18-9 18 9-2zm0 0v-8' },
    { label: 'Receive', path: '/receive', icon: 'M12 5v14m-7-7h14' },
    { label: 'Swap', path: '/swap', icon: 'M7 16V4m0 0L3 8m4-4l4 4m6 4v12m0 0l4-4m-4 4l-4-4' },
    { label: 'Bridge', path: '/bridge', icon: 'M7 7h10M7 7l3-3M7 7l3 3M17 17H7m10 0l-3-3m3 3l-3 3' },
  ];

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Portfolio summary */}
      <div
        className="rounded-2xl p-6 mb-6"
        style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)' }}
      >
        <p className="text-sm uppercase tracking-wide" style={{ color: 'var(--color-text-tertiary)' }}>
          Total Portfolio Value
        </p>
        <p className="mt-1 text-4xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
          ${totalValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
        </p>
        <div className="mt-4 flex gap-3">
          <button
            onClick={refreshBalances}
            className="px-4 py-2 text-sm rounded-lg transition-colors hover:opacity-80"
            style={{ background: 'var(--color-bg-tertiary)', color: 'var(--color-text-primary)' }}
          >
            Refresh
          </button>
          <button
            onClick={() => navigate('/wallet')}
            className="px-4 py-2 text-sm rounded-lg transition-colors hover:opacity-80"
            style={{ background: 'var(--color-primary)', color: '#fff' }}
          >
            Manage Wallets
          </button>
        </div>
      </div>

      {/* Quick actions */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        {quickActions.map((a) => (
          <button
            key={a.path}
            onClick={() => navigate(a.path)}
            className="flex flex-col items-center gap-2 p-4 rounded-xl transition-colors hover:opacity-80"
            style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)' }}
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--color-primary)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d={a.icon} />
            </svg>
            <span className="text-sm" style={{ color: 'var(--color-text-primary)' }}>{a.label}</span>
          </button>
        ))}
      </div>

      {/* Active wallet */}
      {activeWallet && (
        <div
          className="rounded-2xl p-6 mb-6"
          style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)' }}
        >
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold" style={{ color: 'var(--color-text-primary)' }}>
              Active Wallet
            </h2>
            <span
              className="px-3 py-1 text-xs rounded-full"
              style={{ background: 'var(--color-bg-tertiary)', color: 'var(--color-text-secondary)' }}
            >
              {activeWallet.chain?.name || '—'}
            </span>
          </div>
          <p className="font-mono text-sm truncate" style={{ color: 'var(--color-text-secondary)' }}>
            {activeWallet.address}
          </p>
          <p className="mt-2 text-2xl font-bold" style={{ color: 'var(--color-text-primary)' }}>
            {activeWallet.balance || '0'} {activeWallet.chain?.symbol || ''}
          </p>
          <p className="text-sm" style={{ color: 'var(--color-text-tertiary)' }}>
            ≈ ${(activeWallet.balanceUSD || 0).toLocaleString(undefined, { maximumFractionDigits: 2 })}
          </p>
        </div>
      )}

      {/* Recent activity */}
      <div
        className="rounded-2xl p-6"
        style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)' }}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold" style={{ color: 'var(--color-text-primary)' }}>
            Recent Activity
          </h2>
          <button
            onClick={() => navigate('/history')}
            className="text-sm hover:underline"
            style={{ color: 'var(--color-primary)' }}
          >
            View all
          </button>
        </div>

        {loading ? (
          <LoadingSpinner label="Loading activity..." />
        ) : error ? (
          <p className="text-sm" style={{ color: 'var(--color-error)' }}>{error}</p>
        ) : recentTxs.length === 0 ? (
          <p className="text-sm py-8 text-center" style={{ color: 'var(--color-text-tertiary)' }}>
            No recent transactions yet
          </p>
        ) : (
          <div className="space-y-2">
            {recentTxs.map((tx) => (
              <div
                key={tx.hash}
                className="flex items-center justify-between p-3 rounded-lg"
                style={{ background: 'var(--color-bg-secondary)' }}
              >
                <div className="min-w-0">
                  <p className="text-sm font-mono truncate" style={{ color: 'var(--color-text-primary)' }}>
                    {tx.value || '0'} {tx.token || activeWallet?.chain?.symbol || ''}
                  </p>
                  <p className="text-xs font-mono truncate" style={{ color: 'var(--color-text-tertiary)' }}>
                    {tx.hash}
                  </p>
                </div>
                <span
                  className="px-2 py-1 text-xs rounded-full"
                  style={{
                    background:
                      tx.status === 'confirmed'
                        ? 'var(--color-success-light)'
                        : tx.status === 'failed'
                        ? 'var(--color-error-light)'
                        : 'var(--color-bg-tertiary)',
                    color: 'var(--color-text-secondary)',
                  }}
                >
                  {tx.status}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default HomePage;
