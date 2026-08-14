/**
 * TigerWallet Super Admin - Transactions Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Transactions() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showAnalyze, setShowAnalyze] = useState(false);
  const [analysis, setAnalysis] = useState<any | null>(null);
  const [form, setForm] = useState({ user_id: '', tx_type: 'transfer', amount: '', currency: 'USDT', ip_address: '', country: '', device: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getTransactions();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load transactions');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleAnalyze = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      const r = await superAdminApi.analyzeTransaction({
        user_id: form.user_id,
        tx_type: form.tx_type,
        amount: Number(form.amount),
        currency: form.currency,
        ip_address: form.ip_address || undefined,
        country: form.country || undefined,
        device: form.device || undefined,
      });
      setAnalysis(r);
    } catch (err: any) {
      alert(err?.message || 'Analysis failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleUnflag = async (id: string) => {
    setActionLoading(true);
    try {
      await superAdminApi.unflagTransaction(id);
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to unflag transaction');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Transactions</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowAnalyze((s) => !s)}>
          {showAnalyze ? 'Close Analyzer' : 'Analyze Transaction'}
        </button>
      </div>

      {showAnalyze && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Fraud Analysis</h3>
          <form onSubmit={handleAnalyze} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Type</label><select className="input w-full" value={form.tx_type} onChange={(e) => setForm({ ...form, tx_type: e.target.value })}><option>transfer</option><option>withdrawal</option><option>deposit</option><option>trade</option></select></div>
              <div className="form-group flex-1"><label className="text-secondary">Amount</label><input className="input w-full" type="number" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Currency</label><input className="input w-full" value={form.currency} onChange={(e) => setForm({ ...form, currency: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">IP Address</label><input className="input w-full" value={form.ip_address} onChange={(e) => setForm({ ...form, ip_address: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Country</label><input className="input w-full" value={form.country} onChange={(e) => setForm({ ...form, country: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Device</label><input className="input w-full" value={form.device} onChange={(e) => setForm({ ...form, device: e.target.value })} /></div>
            </div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Run Analysis</button>
          </form>
          {analysis && (
            <div className="alert alert-info mt-4">
              <pre className="text-info" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(analysis, null, 2)}</pre>
            </div>
          )}
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No transactions found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>ID</th><th>User</th><th>Type</th><th>Amount</th><th>Currency</th><th>Chain</th><th>Status</th><th>Flagged</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((t) => (
                <tr key={t.id}>
                  <td className="text-secondary">{t.id.slice(0, 8)}...</td>
                  <td className="text-secondary">{t.user_id}</td>
                  <td className="text-primary">{t.type || t.tx_type}</td>
                  <td className="text-primary">{t.amount}</td>
                  <td className="text-primary">{t.currency}</td>
                  <td className="text-secondary">{t.chain}</td>
                  <td><span className={`badge ${t.status === 'completed' ? 'badge-success' : 'badge-info'}`}>{t.status}</span></td>
                  <td>{t.is_suspicious ? <span className="badge badge-error">flagged</span> : <span className="badge badge-neutral">clean</span>}</td>
                  <td>{t.is_suspicious && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleUnflag(t.id)}>Unflag</button>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}
    </div>
  );
}
