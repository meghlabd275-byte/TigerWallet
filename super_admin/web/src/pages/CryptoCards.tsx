/**
 * TigerWallet Super Admin - Crypto Cards Page
 * Full CRUD + status control over the `/api/v1/admin/crypto-cards` routes on
 * the super_admin/go backend (port 8082). Governance records only.
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../context/ThemeContext';
import { cryptoCardsAPI } from '../services/api';

const STATUSES = ['pending', 'active', 'blocked', 'suspended', 'inactive', 'expired'];

export default function CryptoCards() {
  const { resolvedTheme } = useTheme();
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState({
    user_id: '', card_number: '', network: '', currency: '', balance: '',
    daily_limit: '', monthly_limit: '', status: 'pending',
  });

  // Limit + status modals
  const [limitCard, setLimitCard] = useState<any | null>(null);
  const [limitForm, setLimitForm] = useState({ daily_limit: '', monthly_limit: '' });
  const [statusCard, setStatusCard] = useState<any | null>(null);
  const [statusValue, setStatusValue] = useState('pending');

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await cryptoCardsAPI.getAll();
      setItems(res.data || res.cards || res.items || (Array.isArray(res) ? res : []));
    } catch (err: any) {
      setError(err?.message || 'Failed to load crypto cards');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const resetForm = () => {
    setForm({ user_id: '', card_number: '', network: '', currency: '', balance: '', daily_limit: '', monthly_limit: '', status: 'pending' });
    setEditingId(null);
  };

  const openCreate = () => {
    resetForm();
    setShowForm(true);
  };

  const openEdit = (c: any) => {
    setEditingId(c.id);
    setForm({
      user_id: c.user_id ?? '',
      card_number: c.card_number ?? '',
      network: c.network ?? '',
      currency: c.currency ?? '',
      balance: c.balance != null ? String(c.balance) : '',
      daily_limit: c.daily_limit != null ? String(c.daily_limit) : '',
      monthly_limit: c.monthly_limit != null ? String(c.monthly_limit) : '',
      status: c.status ?? 'pending',
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      const payload = {
        user_id: form.user_id || undefined,
        card_number: form.card_number || undefined,
        network: form.network || undefined,
        currency: form.currency || undefined,
        balance: form.balance === '' ? undefined : Number(form.balance),
        daily_limit: form.daily_limit === '' ? undefined : Number(form.daily_limit),
        monthly_limit: form.monthly_limit === '' ? undefined : Number(form.monthly_limit),
        ...(editingId ? {} : { status: form.status || 'pending' }),
      };
      if (editingId) {
        await cryptoCardsAPI.update(editingId, payload);
      } else {
        await cryptoCardsAPI.create(payload as any);
      }
      setShowForm(false);
      resetForm();
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to save crypto card');
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

  const handleDelete = (c: any) => {
    if (!confirm('Delete this crypto card?')) return;
    run(() => cryptoCardsAPI.delete(c.id));
  };

  const handleBlock = (c: any) => {
    const reason = prompt('Reason for blocking (optional):') ?? '';
    run(() => cryptoCardsAPI.block(c.id, reason || undefined));
  };

  const openLimit = (c: any) => {
    setLimitCard(c);
    setLimitForm({
      daily_limit: c.daily_limit != null ? String(c.daily_limit) : '',
      monthly_limit: c.monthly_limit != null ? String(c.monthly_limit) : '',
    });
  };

  const submitLimit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!limitCard) return;
    await run(() =>
      cryptoCardsAPI.setLimit(limitCard.id, {
        daily_limit: limitForm.daily_limit === '' ? undefined : Number(limitForm.daily_limit),
        monthly_limit: limitForm.monthly_limit === '' ? undefined : Number(limitForm.monthly_limit),
      }),
    );
    setLimitCard(null);
  };

  const openStatus = (c: any) => {
    setStatusCard(c);
    setStatusValue(c.status || 'pending');
  };

  const submitStatus = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!statusCard) return;
    await run(() => cryptoCardsAPI.setStatus(statusCard.id, statusValue));
    setStatusCard(null);
  };

  const badgeClass = (s: string) => {
    if (s === 'active') return 'badge-success';
    if (s === 'blocked' || s === 'suspended' || s === 'expired') return 'badge-danger';
    if (s === 'pending') return 'badge-warning';
    return 'badge-secondary';
  };

  return (
    <div className="p-6" data-theme={resolvedTheme}>
      <h1 className="text-2xl font-bold text-primary mb-6">Crypto Cards</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={openCreate}>New Crypto Card</button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">{editingId ? 'Edit Crypto Card' : 'Create Crypto Card'}</h3>
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Card Number</label><input className="input w-full" value={form.card_number} onChange={(e) => setForm({ ...form, card_number: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Network</label><input className="input w-full" value={form.network} onChange={(e) => setForm({ ...form, network: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Currency</label><input className="input w-full" value={form.currency} onChange={(e) => setForm({ ...form, currency: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Balance</label><input className="input w-full" type="number" value={form.balance} onChange={(e) => setForm({ ...form, balance: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Daily Limit</label><input className="input w-full" type="number" value={form.daily_limit} onChange={(e) => setForm({ ...form, daily_limit: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Monthly Limit</label><input className="input w-full" type="number" value={form.monthly_limit} onChange={(e) => setForm({ ...form, monthly_limit: e.target.value })} /></div>
              {!editingId && (
                <div className="form-group flex-1"><label className="text-secondary">Status</label>
                  <select className="input w-full" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
                    {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
              )}
            </div>
            <div className="flex gap-2">
              <button className="btn btn-primary" disabled={actionLoading} type="submit">{editingId ? 'Update' : 'Create'}</button>
              <button className="btn btn-secondary" type="button" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button>
            </div>
          </form>
        </div></div>
      )}

      {limitCard && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Set Limit — {limitCard.card_number || limitCard.id}</h3>
          <form onSubmit={submitLimit} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Daily Limit</label><input className="input w-full" type="number" value={limitForm.daily_limit} onChange={(e) => setLimitForm({ ...limitForm, daily_limit: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Monthly Limit</label><input className="input w-full" type="number" value={limitForm.monthly_limit} onChange={(e) => setLimitForm({ ...limitForm, monthly_limit: e.target.value })} /></div>
            </div>
            <div className="flex gap-2">
              <button className="btn btn-primary" disabled={actionLoading} type="submit">Save Limit</button>
              <button className="btn btn-secondary" type="button" onClick={() => setLimitCard(null)}>Cancel</button>
            </div>
          </form>
        </div></div>
      )}

      {statusCard && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Update Status — {statusCard.card_number || statusCard.id}</h3>
          <form onSubmit={submitStatus} className="flex flex-col gap-3">
            <div className="form-group"><label className="text-secondary">Status</label>
              <select className="input w-full" value={statusValue} onChange={(e) => setStatusValue(e.target.value)}>
                {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div className="flex gap-2">
              <button className="btn btn-primary" disabled={actionLoading} type="submit">Update Status</button>
              <button className="btn btn-secondary" type="button" onClick={() => setStatusCard(null)}>Cancel</button>
            </div>
          </form>
        </div></div>
      )}

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : items.length === 0 ? (
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No crypto cards found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Card Number</th><th>User</th><th>Network</th><th>Currency</th><th>Balance</th><th>Limits (D/M)</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((c) => (
                <tr key={c.id}>
                  <td className="text-primary">{c.card_number ? `${String(c.card_number).slice(0, 6)}...${String(c.card_number).slice(-4)}` : c.id}</td>
                  <td className="text-secondary">{c.user_id || '-'}</td>
                  <td className="text-secondary">{c.network || '-'}</td>
                  <td className="text-secondary">{c.currency || '-'}</td>
                  <td className="text-secondary">{c.balance ?? '-'}</td>
                  <td className="text-secondary">{c.daily_limit ?? '-'}/{c.monthly_limit ?? '-'}</td>
                  <td><span className={`badge ${badgeClass(c.status)}`}>{c.status || 'unknown'}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => openEdit(c)}>Edit</button>
                    {c.status === 'active' ? (
                      <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleBlock(c)}>Block</button>
                    ) : (
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => cryptoCardsAPI.activate(c.id))}>Activate</button>
                    )}
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => openLimit(c)}>Limit</button>
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => openStatus(c)}>Status</button>
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => handleDelete(c)}>Delete</button>
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
