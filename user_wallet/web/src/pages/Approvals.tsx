// Token Approvals Page — view + revoke ERC-20 token approvals (security).
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' },
  { id: 10, label: 'Optimism' },
  { id: 8453, label: 'Base' },
];

interface Approval { id: string; token?: string; spender?: string; amount?: string; approved?: boolean; }

export default function Approvals() {
  const [address, setAddress] = useState('');
  const [chainId, setChainId] = useState(1);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api.getWallets().then((data) => {
      if (data.wallets && data.wallets.length > 0) { setAddress(data.wallets[0].address); setChainId(data.wallets[0].chain_id); }
    }).catch(() => {});
  }, []);

  const load = async () => {
    if (!address) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.getApprovals(address, chainId);
      setApprovals((data.approvals as Approval[]) || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load approvals');
      setApprovals([]);
    } finally {
      setLoading(false);
    }
  };

  // load() is recreated per render; keying on the inputs it actually reads keeps
  // the fetch from looping.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { if (address) load(); }, [address, chainId]);

  const revoke = async (id: string) => {
    if (!window.confirm('Revoke this approval?')) return;
    setBusy(true);
    try { await api.revokeApproval({ approvalId: id }); load(); } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Revoke failed'); } finally { setBusy(false); }
  };

  return (
    <div className="approvals-page">
      <h1>Token Approvals</h1>
      <p className="hint">Review and revoke ERC-20 spending approvals granted to contracts.</p>
      {error && <div className="error">{error}</div>}
      <div className="filter-row">
        <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
          {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
        </select>
        <input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="0x…" className="address-input" />
      </div>
      {loading ? <p>Loading…</p> : approvals.length === 0 ? <p>No active approvals.</p> : (
        <ul className="approval-list">
          {approvals.map((a) => (
            <li key={a.id}>
              <div className="mono small">{a.token || 'token'}</div>
              <div className="mono small">Spender: {a.spender}</div>
              <div className="small">Amount: {a.amount}</div>
              <button onClick={() => revoke(a.id)} disabled={busy}>Revoke</button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
