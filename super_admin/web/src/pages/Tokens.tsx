/**
 * TigerWallet Super Admin - Tokens Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Tokens() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    address: '', name: '', symbol: '', decimals: '18', chain: '', chain_id: '', total_supply: '', logo_url: '', website_url: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getTokens();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load tokens');
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
      await superAdminApi.createToken({
        address: form.address,
        name: form.name,
        symbol: form.symbol,
        decimals: Number(form.decimals),
        chain: form.chain,
        chain_id: Number(form.chain_id),
        total_supply: Number(form.total_supply),
        logo_url: form.logo_url || undefined,
        website_url: form.website_url || undefined,
      });
      setShowForm(false);
      setForm({ address: '', name: '', symbol: '', decimals: '18', chain: '', chain_id: '', total_supply: '', logo_url: '', website_url: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create token');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAction = async (address: string, action: 'pause' | 'unpause' | 'verify' | 'delete') => {
    if (action === 'delete' && !confirm('Delete this token?')) return;
    setActionLoading(true);
    try {
      if (action === 'pause') await superAdminApi.pauseToken(address);
      else if (action === 'unpause') await superAdminApi.unpauseToken(address);
      else if (action === 'verify') await superAdminApi.verifyToken(address);
      else await superAdminApi.deleteToken(address);
      load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Tokens</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Token'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6">
          <div className="card-body">
            <h3 className="text-primary mb-4">Create Token</h3>
            <form onSubmit={handleCreate} className="flex flex-col gap-3">
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">Address</label><input className="input w-full" value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Symbol</label><input className="input w-full" value={form.symbol} onChange={(e) => setForm({ ...form, symbol: e.target.value })} required /></div>
              </div>
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">Decimals</label><input className="input w-full" type="number" value={form.decimals} onChange={(e) => setForm({ ...form, decimals: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Chain</label><input className="input w-full" value={form.chain} onChange={(e) => setForm({ ...form, chain: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Chain ID</label><input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Total Supply</label><input className="input w-full" type="number" value={form.total_supply} onChange={(e) => setForm({ ...form, total_supply: e.target.value })} required /></div>
              </div>
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">Logo URL</label><input className="input w-full" value={form.logo_url} onChange={(e) => setForm({ ...form, logo_url: e.target.value })} /></div>
                <div className="form-group flex-1"><label className="text-secondary">Website URL</label><input className="input w-full" value={form.website_url} onChange={(e) => setForm({ ...form, website_url: e.target.value })} /></div>
              </div>
              <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
            </form>
          </div>
        </div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No tokens found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Symbol</th><th>Name</th><th>Chain</th><th>Address</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((t) => (
                <tr key={t.address || t.id}>
                  <td className="text-primary">{t.symbol}</td>
                  <td className="text-primary">{t.name}</td>
                  <td className="text-secondary">{t.chain}</td>
                  <td className="text-secondary">{t.address ? `${t.address.slice(0, 8)}...` : '-'}</td>
                  <td><span className={`badge ${t.status === 'active' ? 'badge-success' : 'badge-warning'}`}>{t.status || 'unknown'}</span></td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-primary" disabled={actionLoading} onClick={() => handleAction(t.address, 'verify')}>Verify</button>
                    {t.status === 'paused' ? (
                      <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleAction(t.address, 'unpause')}>Unpause</button>
                    ) : (
                      <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleAction(t.address, 'pause')}>Pause</button>
                    )}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => handleAction(t.address, 'delete')}>Delete</button>
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
