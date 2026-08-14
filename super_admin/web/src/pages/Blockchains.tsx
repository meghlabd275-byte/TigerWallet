/**
 * TigerWallet Super Admin - Blockchains Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Blockchains() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    name: '', symbol: '', chain_id: '', is_evm: 'true', rpc_url: '', explorer_url: '', native_token: '', decimals: '18', avg_gas_price_gwei: '', block_time_seconds: '', logo_url: '',
  });

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const res: any = await superAdminApi.getBlockchains();
      setItems(res.data || res.items || res || []);
    } catch (err: any) {
      setError(err?.message || 'Failed to load blockchains');
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
      await superAdminApi.createBlockchain({
        name: form.name,
        symbol: form.symbol,
        chain_id: Number(form.chain_id),
        is_evm: form.is_evm === 'true',
        rpc_url: form.rpc_url,
        explorer_url: form.explorer_url,
        native_token: form.native_token,
        decimals: Number(form.decimals),
        avg_gas_price_gwei: form.avg_gas_price_gwei ? Number(form.avg_gas_price_gwei) : undefined,
        block_time_seconds: form.block_time_seconds ? Number(form.block_time_seconds) : undefined,
        logo_url: form.logo_url || undefined,
      });
      setShowForm(false);
      setForm({ name: '', symbol: '', chain_id: '', is_evm: 'true', rpc_url: '', explorer_url: '', native_token: '', decimals: '18', avg_gas_price_gwei: '', block_time_seconds: '', logo_url: '' });
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to create blockchain');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAction = async (id: string, action: 'activate' | 'deactivate' | 'test' | 'delete') => {
    if (action === 'delete' && !confirm('Delete this blockchain?')) return;
    setActionLoading(true);
    try {
      if (action === 'activate') await superAdminApi.activateBlockchain(id);
      else if (action === 'deactivate') await superAdminApi.deactivateBlockchain(id);
      else if (action === 'test') {
        const r = await superAdminApi.testBlockchainRPC(id);
        alert(`RPC test: ${r.success ? 'OK' : 'Failed'} | latency ${r.latency_ms}ms | block ${r.block_number}`);
      } else await superAdminApi.deleteBlockchain(id);
      if (action !== 'test') load();
    } catch (err: any) {
      alert(err?.message || 'Action failed');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Blockchains</h1>

      <div className="flex justify-between items-center mb-4">
        <button className="btn btn-primary" onClick={() => setShowForm((s) => !s)}>
          {showForm ? 'Cancel' : 'New Blockchain'}
        </button>
      </div>

      {showForm && (
        <div className="card mb-6">
          <div className="card-body">
            <h3 className="text-primary mb-4">Create Blockchain</h3>
            <form onSubmit={handleCreate} className="flex flex-col gap-3">
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">Name</label><input className="input w-full" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Symbol</label><input className="input w-full" value={form.symbol} onChange={(e) => setForm({ ...form, symbol: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Chain ID</label><input className="input w-full" type="number" value={form.chain_id} onChange={(e) => setForm({ ...form, chain_id: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">EVM</label><select className="input w-full" value={form.is_evm} onChange={(e) => setForm({ ...form, is_evm: e.target.value })}><option value="true">Yes</option><option value="false">No</option></select></div>
              </div>
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">RPC URL</label><input className="input w-full" value={form.rpc_url} onChange={(e) => setForm({ ...form, rpc_url: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Explorer URL</label><input className="input w-full" value={form.explorer_url} onChange={(e) => setForm({ ...form, explorer_url: e.target.value })} required /></div>
              </div>
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">Native Token</label><input className="input w-full" value={form.native_token} onChange={(e) => setForm({ ...form, native_token: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Decimals</label><input className="input w-full" type="number" value={form.decimals} onChange={(e) => setForm({ ...form, decimals: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Avg Gas (gwei)</label><input className="input w-full" type="number" value={form.avg_gas_price_gwei} onChange={(e) => setForm({ ...form, avg_gas_price_gwei: e.target.value })} /></div>
                <div className="form-group flex-1"><label className="text-secondary">Block Time (s)</label><input className="input w-full" type="number" value={form.block_time_seconds} onChange={(e) => setForm({ ...form, block_time_seconds: e.target.value })} /></div>
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
        <div className="card"><div className="card-body text-center py-8"><p className="text-secondary">No blockchains found.</p></div></div>
      ) : (
        <div className="card"><div className="card-body overflow-x-auto">
          <table className="table"><thead><tr><th>Name</th><th>Symbol</th><th>Chain ID</th><th>Native Token</th><th>Active</th><th>Actions</th></tr></thead>
            <tbody>
              {items.map((b) => (
                <tr key={b.id}>
                  <td className="text-primary">{b.name}</td>
                  <td className="text-secondary">{b.symbol}</td>
                  <td className="text-secondary">{b.chain_id}</td>
                  <td className="text-secondary">{b.native_token}</td>
                  <td><span className={`badge ${b.is_active ? 'badge-success' : 'badge-neutral'}`}>{b.is_active ? 'active' : 'inactive'}</span></td>
                  <td><div className="flex gap-2">
                    <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleAction(b.id, 'test')}>Test RPC</button>
                    {b.is_active ? (
                      <button className="btn btn-secondary" disabled={actionLoading} onClick={() => handleAction(b.id, 'deactivate')}>Deactivate</button>
                    ) : (
                      <button className="btn btn-primary" disabled={actionLoading} onClick={() => handleAction(b.id, 'activate')}>Activate</button>
                    )}
                    <button className="btn btn-danger" disabled={actionLoading} onClick={() => handleAction(b.id, 'delete')}>Delete</button>
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
