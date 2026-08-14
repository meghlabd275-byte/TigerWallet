/**
 * TigerWallet Super Admin - Bots Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Bots() {
  const [items, setItems] = useState<any[]>([]);
  const [tiers, setTiers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [showTierForm, setShowTierForm] = useState(false);
  const [form, setForm] = useState({ name: '', bot_type: '' });
  const [tierForm, setTierForm] = useState({ name: '', bot_id: '', level: '', min_volume: '', max_volume: '', fee_percent: '' });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const [botRes, tierRes]: any = await Promise.all([
        superAdminApi.getBots(),
        superAdminApi.getBotTiers(),
      ]);
      setItems(botRes.bots || botRes || []);
      setTiers(tierRes.tiers || tierRes || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load bots');
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
      await superAdminApi.createBot({ name: form.name, bot_type: form.bot_type });
      setShowForm(false);
      setForm({ name: '', bot_type: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create bot');
    } finally {
      setActionLoading(false);
    }
  };

  const handleCreateTier = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      await superAdminApi.createBotTier({
        name: tierForm.name,
        bot_id: tierForm.bot_id || undefined,
        level: tierForm.level ? Number(tierForm.level) : undefined,
        min_volume: tierForm.min_volume ? Number(tierForm.min_volume) : undefined,
        max_volume: tierForm.max_volume ? Number(tierForm.max_volume) : undefined,
        fee_percent: tierForm.fee_percent ? Number(tierForm.fee_percent) : undefined,
      });
      setShowTierForm(false);
      setTierForm({ name: '', bot_id: '', level: '', min_volume: '', max_volume: '', fee_percent: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create bot tier');
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
      <h1 className="text-2xl font-bold text-primary mb-6">Bots</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Bot'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Bot</h3>
          <form onSubmit={handleCreate} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Bot Type</label><input className="input w-full" value={form.bot_type} onChange={(e) => setForm({ ...form, bot_type: e.target.value })} required /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No bots found.</p></div></div>
      ) : (
        <div className="card mb-6"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Bot Type</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((b) => (
                <tr key={b.id}>
                  <td className="text-primary">{b.name}</td>
                  <td className="text-secondary">{b.bot_type}</td>
                  <td><span className={`badge ${b.status === 'active' ? 'badge-success' : 'badge-neutral'}`}>{b.status || 'inactive'}</span></td>
                  <td><div className="flex gap-2" style={{ flexWrap: 'wrap' }}>
                    {b.status !== 'active' && <button className="btn btn-primary" disabled={actionLoading} onClick={() => run(() => superAdminApi.updateBotStatus(b.id, 'active'))}>Start</button>}
                    {b.status === 'active' && <button className="btn btn-secondary" disabled={actionLoading} onClick={() => run(() => superAdminApi.updateBotStatus(b.id, 'inactive'))}>Stop</button>}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this bot?')) run(() => superAdminApi.deleteBot(b.id)); }}>Delete</button>
                  </div></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div></div>
      )}

      <div className="flex justify-between items-center mb-4 mt-8">
        <h2 className="text-xl font-bold text-primary">Bot Tiers</h2>
        <button className="btn btn-primary" onClick={() => setShowTierForm((s) => !s)}>
          {showTierForm ? 'Cancel' : 'New Tier'}
        </button>
      </div>

      {showTierForm && (
        <div className="card mb-6"><div className="card-body">
          <h3 className="text-primary mb-4">Create Bot Tier</h3>
          <form onSubmit={handleCreateTier} className="flex flex-col gap-3">
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={tierForm.name} onChange={(e) => setTierForm({ ...tierForm, name: e.target.value })} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Bot ID</label><input className="input w-full" value={tierForm.bot_id} onChange={(e) => setTierForm({ ...tierForm, bot_id: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Level</label><input className="input w-full" type="number" value={tierForm.level} onChange={(e) => setTierForm({ ...tierForm, level: e.target.value })} /></div>
            </div>
            <div className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">Min Volume</label><input className="input w-full" type="number" value={tierForm.min_volume} onChange={(e) => setTierForm({ ...tierForm, min_volume: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Max Volume</label><input className="input w-full" type="number" value={tierForm.max_volume} onChange={(e) => setTierForm({ ...tierForm, max_volume: e.target.value })} /></div>
              <div className="form-group flex-1"><label className="text-secondary">Fee Percent</label><input className="input w-full" type="number" value={tierForm.fee_percent} onChange={(e) => setTierForm({ ...tierForm, fee_percent: e.target.value })} /></div>
            </div>
            <button className="btn btn-primary" disabled={actionLoading} type="submit">Create Tier</button>
          </form>
        </div></div>
      )}

      {!error && !loading && (
        tiers.length === 0 ? (
          <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No bot tiers found.</p></div></div>
        ) : (
          <div className="card"><div className="card-body overflow-x-auto">
            <table className="table"><thead><tr><th>Name</th><th>Level</th><th>Min Volume</th><th>Max Volume</th><th>Fee</th><th>Actions</th></tr></thead>
              <tbody>
                {tiers.map((t) => (
                  <tr key={t.id}>
                    <td className="text-primary">{t.name}</td>
                    <td className="text-secondary">{t.level ?? '-'}</td>
                    <td className="text-secondary">{t.min_volume ?? '-'}</td>
                    <td className="text-secondary">{t.max_volume ?? '-'}</td>
                    <td className="text-secondary">{t.fee_percent ?? '-'}%</td>
                    <td><div className="flex gap-2">
                      <button className="btn btn-danger" disabled={actionLoading} onClick={() => { if (confirm('Delete this tier?')) run(() => superAdminApi.deleteBotTier(t.id)); }}>Delete</button>
                    </div></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div></div>
        )
      )}
    </div>
  );
}
