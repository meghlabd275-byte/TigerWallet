/**
 * TigerWallet Super Admin - Trading Pairs Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function TradingPairs() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ base: '', quote: '', chain_id: '', min_trade_amount: '', max_trade_amount: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getTradingPairs();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load trading pairs');
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
      await superAdminApi.createTradingPair({
        base: form.base,
        quote: form.quote,
        chain_id: Number(form.chain_id),
        min_trade_amount: form.min_trade_amount ? Number(form.min_trade_amount) : undefined,
        max_trade_amount: form.max_trade_amount ? Number(form.max_trade_amount) : undefined,
      });
      setShowForm(false);
      setForm({ base: '', quote: '', chain_id: '', min_trade_amount: '', max_trade_amount: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create trading pair');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAction = async (id: string, action: 'suspend' | 'resume' | 'delete') => {
    if (action === 'delete' && !confirm('Delete this trading pair?')) return;
    setActionLoading(true);
    try {
      if (action === 'suspend') await superAdminApi.suspendTradingPair(id);
      else if (action === 'resume') await superAdminApi.resumeTradingPair(id);
      else await superAdminApi.deleteTradingPair(id);
      load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Trading Pairs</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Pair'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6">
          <div className="card-body">
            <h3 className="text-primary mb-4">Create Trading Pair</h3>
            <form onSubmit={handleCreate} className="flex flex-col gap-3">
              <div className="flex gap-3">
                <div className="form-group flex-1">
                  <label className="text-secondary">Base</label>
                  <input className="input w-full" value={form.base} onChange={(e) => setForm({ ...form, base: e.target.value })} required />
                </div>
                <div className="form-group flex-1">
                  <label className="text-secondary">Quote</label>
                  <input className="input w-full" value={form.quote} onChange={(e) => setForm({ ...form, quote: e.target.value })} required />
                </div>
                <div className="form-group flex-1">
                  <label className="text-secondary">Chain ID</label>
                  <input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} required />
                </div>
              </div>
              <div className="flex gap-3">
                <div className="form-group flex-1">
                  <label className="text-secondary">Min Trade Amount</label>
                  <input className="input w-full" type="number" value={form.min_trade_amount} onChange={(e) => setForm({ ...form, min_trade_amount: e.target.value })} />
                </div>
                <div className="form-group flex-1">
                  <label className="text-secondary">Max Trade Amount</label>
                  <input className="input w-full" type="number" value={form.max_trade_amount} onChange={(e) => setForm({ ...form, max_trade_amount: e.target.value })} />
                </div>
              </div>
              <button className="btn btn-primary" disabled={actionLoading} type="submit">Create</button>
            </form>
          </div>
        </div>
      )}

      {error ? (
        <div className="alert alert-error mb-4">
          <p className="text-error">{error}</p>
          <button className="btn btn-secondary mt-2" onClick={load}>Retry</button>
        </div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No trading pairs found.</p></div></div>
      ) : (
        <div className="card">
          <div className="card-body overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>Pair</th><th>Chain</th><th>Price</th><th>24h Volume</th><th>Status</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((p) => (
                  <tr key={p.id}>
                    <td className="text-primary">{p.pair_name || `${p.base}/${p.quote}`}</td>
                    <td className="text-secondary">{p.chain || p.chain_id}</td>
                    <td className="text-primary">{p.price ?? '-'}</td>
                    <td className="text-secondary">{p.volume_24h ?? '-'}</td>
                    <td>
                      <span className={`badge ${p.status === 'active' ? 'badge-success' : 'badge-warning'}`}>{p.status}</span>
                    </td>
                    <td>
                      <div className="flex gap-2">
                        {p.status === 'suspended' || p.status === 'halted' ? (
                          <button className="btn btn-primary" disabled={actionLoading} onClick={() => handleAction(p.id, 'resume')}>Resume</button>
                        ) : (
                          <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleAction(p.id, 'suspend')}>Suspend</button>
                        )}
                        <button className="btn btn-danger" disabled={actionLoading} onClick={() => handleAction(p.id, 'delete')}>Delete</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
