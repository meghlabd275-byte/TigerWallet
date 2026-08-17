/**
 * Approvals Page - ERC-20 token approvals.
 *
 * Fetches active approvals for the active wallet from the canonical backend
 * (GET /approvals?address=&chain_id=) and lets the user revoke each
 * (DELETE /approvals/:id). All calls go through WalletService; no mock data.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useWallet } from '../contexts/WalletContext';
import { WalletService } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

interface Approval {
  id: string;
  spender?: string;
  token?: string;
  token_symbol?: string;
  amount?: string;
  chain_id?: number;
}

function ApprovalsPage() {
  const { theme } = useTheme();
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());

  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const address = activeWallet?.address ?? '';
  const chainId = activeWallet?.chain?.chainId ?? 1;

  const loadApprovals = useCallback(async () => {
    if (!address) {
      setApprovals([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = (await walletService.getApprovals(address, chainId)) as
        | Approval[]
        | { approvals?: Approval[] };
      const list = Array.isArray(data) ? data : (data?.approvals ?? []);
      setApprovals(
        (list ?? []).map((a, i) => ({
          id: String(a.id ?? i),
          spender: a.spender ? String(a.spender) : undefined,
          token: a.token ? String(a.token) : undefined,
          token_symbol: a.token_symbol ? String(a.token_symbol) : undefined,
          amount: a.amount ? String(a.amount) : undefined,
          chain_id: typeof a.chain_id === 'number' ? a.chain_id : undefined,
        }))
      );
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load approvals');
      setApprovals([]);
    } finally {
      setLoading(false);
    }
  }, [walletService, address, chainId]);

  useEffect(() => {
    loadApprovals();
  }, [loadApprovals]);

  const handleRevoke = async (id: string) => {
    setError(null);
    setRevokingId(id);
    try {
      await walletService.revokeApproval({ approvalId: id });
      await loadApprovals();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to revoke approval');
    } finally {
      setRevokingId(null);
    }
  };

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Token Approvals</h1>

      {!activeWallet && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          <p className="text-sm opacity-60">Select a wallet to view its token approvals.</p>
        </div>
      )}

      {error && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <p className="text-sm text-red-500">{error}</p>
        </div>
      )}

      {loading ? (
        <LoadingSpinner label="Loading approvals..." />
      ) : approvals.length === 0 ? (
        <div className={`card text-center py-12 opacity-60 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          No active token approvals.
        </div>
      ) : (
        <div className="space-y-3">
          {approvals.map((a) => (
            <div key={a.id} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start gap-3">
                <div className="min-w-0">
                  <div className="font-semibold">
                    {a.token_symbol || a.token || 'Unknown token'}
                  </div>
                  {a.spender && (
                    <p className="text-xs font-mono opacity-60 mt-1 truncate">Spender: {a.spender}</p>
                  )}
                  {a.token && (
                    <p className="text-xs font-mono opacity-40 mt-1 truncate">Token: {a.token}</p>
                  )}
                  <div className="flex gap-3 mt-1">
                    {a.amount && (
                      <span className="text-xs opacity-60">Amount: {a.amount}</span>
                    )}
                    {a.chain_id !== undefined && (
                      <span className="text-xs opacity-40">Chain: {a.chain_id}</span>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => handleRevoke(a.id)}
                  disabled={revokingId === a.id}
                  className="btn btn-secondary text-sm whitespace-nowrap"
                >
                  {revokingId === a.id ? 'Revoking...' : 'Revoke'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default ApprovalsPage;
