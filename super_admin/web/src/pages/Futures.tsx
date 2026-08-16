/**
 * TigerWallet Super Admin - Futures Positions Page
 * Governance records only — never moves crypto assets.
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

const STATUSES = ['open', 'closed', 'liquidated', 'paused', 'halted'];

export default function Futures() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    pair: '', side: 'long', size: '', leverage: '', entry_price: '',
    liquidation_price: '', margin: '', chain_id: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getFutures();
      setItems(res.positions || res.data || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load futures positions');
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
      await superAdminApi.createFuture({
        pair: form.pair,
        side: form.side,
        size: form.size ? Number(form.size) : undefined,
        leverage: form.leverage ? Number(form.leverage) : undefined,
        entry_price: form.entry_price ? Number(form.entry_price) : undefined,
        liquidation_price: form.liquidation_price ? Number(form.liquidation_price) : undefined,
        margin: form.margin ? Number(form.margin) : undefined,
        chain_id: form.chain_id ? Number(form.chain_id) : undefined,
      });
      setShowForm(false);
      setForm({ pair: '', side: 'long', size: '', leverage: '', entry_price: '', liquidation_price: '', margin: '', chain_id: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create futures position');
    } finally {
      setActionLoading(false);
    }
  };

  const run = async (fn: () => Promise<any>) => {
    setActionLoading(true);
    try {
      await fn();
      load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Futures Positions</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Position'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Futures Position</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Pair</label><input className="input w-full" value={form.pair} onChange={(e) => setForm({ ...form, pair: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Side</label><select className="input w-full" value={form.side} onChange={(e) => setForm({ ...form, side: e.target.value })}><option>long</option><option>short</option></select></div>
              <div className="form-group flex-1"><label className="text-secondary">Size</label><input className="input w-full" type="number" step="any" value={form.size} onChange={(e) => setForm({ ...form, size: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Leverage</label><input className="input w-full" type="number" step="any" value={form.leverage} onChange={(e) => setForm({ ...form, leverage: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Entry Price</label><input className="input w-full" type="number" step="any" value={form.entry_price} onChange={(e) => setForm({ ...form, entry_price: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Liquidation Price</label><input className="input w-full" type="number" step="any" value={form.liquidation_price} onChange={(e) => setForm({ ...form, liquidation_price: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Margin</label><input className="input w-full" type="number" step="any" value={form.margin} onChange={(e) => setForm({ ...form, margin: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Chain ID</label><input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No futures positions found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Pair</th><th>Side</th><th>Size</th><th>Leverage</th><th>Entry Price</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((p) => (
                <tr key={p.id}>
                  <td className="text-primary">{p.pair}</td>
                  <td className="text-secondary">{p.side}</td>
                  <td className="text-secondary">{p.size}</td>
                  <td className="text-secondary">{p.leverage}</td>
                  <td className="text-secondary">{p.entry_price}</td>
                  <td><span className={`badge ${p.status === 'open' ? 'badge-success' : p.status === 'liquidated' ? 'badge-error' : 'badge-warning'}`}>{p.status}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <select className="input" value={p.status || ''} disabled={actionLoading} onChange={(e) => run(() => superAdminApi.updateFutureStatus(p.id, e.target.value))}>
                      {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                    </select>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this position?')) run(() => superAdminApi.deleteFuture(p.id)); }}>Delete</button>
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
