/**
 * TigerWallet Super Admin - Master Wallets Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function MasterWallets() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', address: '', chain_id: '' });
  const [balances, setBalances] = useState<Record<string, number | null>>({});
  const [transferTarget, setTransferTarget] = useState<any | null>(null);
  const [transferForm, setTransferForm] = useState({ amount: '', to_wallet_id: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getMasterWallets();
      setItems(res.wallets || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load master wallets');
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
      await superAdminApi.createMasterWallet({
        name: form.name,
        address: form.address || undefined,
        chain_id: form.chain_id ? Number(form.chain_id) : undefined,
      });
      setShowForm(false);
      setForm({ name: '', address: '', chain_id: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create master wallet');
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
      const r: any = await superAdminApi.getMasterWalletBalance(w.id);
      setBalances((prev) => ({ ...prev, [w.id]: r.balance }));
    }, false);
  };

  const handleTransfer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!transferTarget) return;
    await run(async () => {
      await superAdminApi.masterWalletTransfer(transferTarget.id, {
        amount: Number(transferForm.amount),
        to_wallet_id: transferForm.to_wallet_id,
      });
    });
    setTransferTarget(null);
    setTransferForm({ amount: '', to_wallet_id: '' });
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Master Wallets</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Master Wallet'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Master Wallet</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Chain ID</label><input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} /></div>
            </div>
            <div className="form-group"><label className="text-secondary">Address</label><input className="input w-full" value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} /></div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No master wallets found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Address</th><th>Chain ID</th><th>Balance</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((w) => (
                <tr key={w.id}>
                  <td className="text-primary">{w.name}</td>
                  <td className="text-secondary">{w.address ? `${w.address.slice(0, 10)}...` : '-'}</td>
                  <td className="text-secondary">{w.chain_id ?? '-'}</td>
                  <td className="text-secondary">{balances[w.id] != null ? balances[w.id] : '-'}</td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => fetchBalance(w)}>Balance</button>
                    <button className="btn btn-primary" disabled={actionLoading} onClick={() => { setTransferTarget(w); setTransferForm({ amount: '', to_wallet_id: '' }); }}>Transfer</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this wallet?')) run(() => superAdminApi.deleteMasterWallet(w.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      {transferTarget && (
        <div className="card mt-4"><div className="card-body">
          <h3 className="text-primary mb-2">Transfer from {transferTarget.name}</h3>
          <form onSubmit={handleTransfer} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Amount</label><input className="input w-full" type="number" value={transferForm.amount} onChange={(e) => setTransferForm({ ...transferForm, amount: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">To Wallet ID</label><input className="input w-full" value={transferForm.to_wallet_id} onChange={(e) => setTransferForm({ ...transferForm, to_wallet_id: e.target.value })} required /></div>
            </div>
            <div className="flex gap-2"><button className="btn btn-primary" disabled={actionLoading} type="submit">Transfer</button><button className="btn btn-secondary" type="button" onClick={() => setTransferTarget(null)}>Cancel</button></div>
          </form>
        </div></div>
      )}
    </div>
  );
}
