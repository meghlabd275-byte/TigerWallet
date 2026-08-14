/**
 * TigerWallet Super Admin - User Wallets Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function UserWallets() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', master_wallet_id: '', address: '', chain_id: '' });
  const [balances, setBalances] = useState<Record<string, number | null>>({});

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getUserWallets();
      setItems(res.wallets || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load user wallets');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createUserWallet({
        name: form.name,
        master_wallet_id: form.master_wallet_id || undefined,
        address: form.address || undefined,
        chain_id: form.chain_id ? Number(form.chain_id) : undefined,
      });
      setShowForm(false);
      setForm({ name: '', master_wallet_id: '', address: '', chain_id: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create user wallet');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>, reload = true) => {
    setActionLoading(true);
    try {
      await fn();
      if (reload) load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  const fetchBalance = (w: any) => {
    run(async () => {
      const r: any = await superAdminApi.getUserWalletBalance(w.id);
      setBalances((prev) => ({ ...prev, [w.id]: r.balance }));
    }, false);
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">User Wallets</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New User Wallet'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create User Wallet</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Master Wallet ID</label><input className="input w-full" value={form.master_wallet_id} onChange={(e) => setForm({ ...form, master_wallet_id: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Chain ID</label><input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Address</label><input className="input w-full" value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} /></div>
            </div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No user wallets found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Address</th><th>Chain ID</th><th>Master Wallet</th><th>Balance</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td className="text-primary">{w.name}</td>
                  <td className="text-secondary">{w.address ? `${w.address.slice(0, 10)}...` : '-'}</td>
                  <td className="text-secondary">{w.chain_id ?? '-'}</td>
                  <td className="text-secondary">{w.master_wallet_id || '-'}</td>
                  <td className="text-secondary">{balances[w.id] != null ? balances[w.id] : '-'}</td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => fetchBalance(w)}>Balance</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this wallet?')) run(() => superAdminApi.deleteUserWallet(w.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}
    </div>
  );
}
