// Tokens Page — WL-ProjectParty. Full token CRUD against the real backend:
// POST /tokens (create), GET /tokens (list), GET /tokens/:id (get),
// PUT /tokens/:id (update), DELETE /tokens/:id (delete).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const STATUSES = ['draft', 'pending', 'active', 'rejected'];
const LISTING_TYPES = ['standard', 'premium', 'featured'];
const CHAINS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' }
];

interface TokenForm {
  name: string; symbol: string; contract_address: string; chain_id: number;
  decimals: number; logo_url: string; description: string; website: string;
  status: string; listing_type: string;
}

const EMPTY_FORM: TokenForm = {
  name: '', symbol: '', contract_address: '', chain_id: 1,
  decimals: 18, logo_url: '', description: '', website: '',
  status: 'draft', listing_type: 'standard'
};

export default function Tokens() {
  const [tokens, setTokens] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<TokenForm>(EMPTY_FORM);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [detail, setDetail] = useState<any | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.listTokens(statusFilter || undefined);
      setTokens(data.tokens || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load tokens');
    }
    setLoading(false);
  }, [statusFilter]);

  useEffect(() => { load(); }, [load]);

  const resetForm = () => {
    setForm(EMPTY_FORM);
    setEditingId(null);
    setShowForm(false);
  };

  const startCreate = () => {
    setForm(EMPTY_FORM);
    setEditingId(null);
    setShowForm(true);
    setMsg(null);
  };

  const startEdit = (t: any) => {
    setForm({
      name: t.name || '', symbol: t.symbol || '', contract_address: t.contract_address || '',
      chain_id: t.chain_id ?? 1, decimals: t.decimals ?? 18, logo_url: t.logo_url || '',
      description: t.description || '', website: t.website || '',
      status: t.status || 'draft', listing_type: t.listing_type || 'standard'
    });
    setEditingId(t.id);
    setShowForm(true);
    setMsg(null);
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      if (editingId) {
        await api.updateToken(editingId, form);
        setMsg({ type: 'success', text: 'Token updated.' });
      } else {
        await api.createToken(form);
        setMsg({ type: 'success', text: 'Token created.' });
      }
      resetForm();
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to save token' });
    }
    setSubmitting(false);
  };

  const remove = async (id: string) => {
    if (!window.confirm('Delete this token? This cannot be undone.')) return;
    try {
      await api.deleteToken(id);
      setMsg({ type: 'success', text: 'Token deleted.' });
      if (detail?.id === id) setDetail(null);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to delete token' });
    }
  };

  const view = async (id: string) => {
    setMsg(null);
    try {
      const data = await api.getToken(id);
      setDetail(data);
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to load token' });
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Tokens</h1>
        <div className="row-actions">
          <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}>
            <option value="">All statuses</option>
            {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
          <button onClick={startCreate}>{showForm && !editingId ? 'Close' : 'Create Token'}</button>
        </div>
      </div>
      <p className="subtitle">Create, update, and delete tokens stored in the WL-ProjectParty backend.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>{editingId ? 'Update Token' : 'Create Token'}</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Name</label>
                <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Symbol</label>
                <input value={form.symbol} onChange={e => setForm({ ...form, symbol: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Contract Address</label>
                <input value={form.contract_address} onChange={e => setForm({ ...form, contract_address: e.target.value })} placeholder="0x…" />
              </div>
              <div className="form-field">
                <label>Chain</label>
                <select value={form.chain_id} onChange={e => setForm({ ...form, chain_id: parseInt(e.target.value) })}>
                  {CHAINS.map(c => <option key={c.id} value={c.id}>{c.label}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Decimals</label>
                <input type="number" value={form.decimals} onChange={e => setForm({ ...form, decimals: parseInt(e.target.value) || 0 })} />
              </div>
              <div className="form-field">
                <label>Logo URL</label>
                <input value={form.logo_url} onChange={e => setForm({ ...form, logo_url: e.target.value })} placeholder="https://…" />
              </div>
              <div className="form-field">
                <label>Website</label>
                <input value={form.website} onChange={e => setForm({ ...form, website: e.target.value })} placeholder="https://…" />
              </div>
              <div className="form-field">
                <label>Status</label>
                <select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>
                  {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Listing Type</label>
                <select value={form.listing_type} onChange={e => setForm({ ...form, listing_type: e.target.value })}>
                  {LISTING_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div className="form-field" style={{ gridColumn: '1 / -1' }}>
                <label>Description</label>
                <textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Saving…' : (editingId ? 'Update' : 'Create')}</button>
              <button type="button" className="secondary" onClick={resetForm}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {detail && (
        <section>
          <div className="section-title">
            <h2>Token Detail</h2>
            <button className="secondary" onClick={() => setDetail(null)}>Close</button>
          </div>
          <div className="cards-grid">
            <div className="card">
              {detail.logo_url && <img src={detail.logo_url} alt={detail.name} style={{ width: 48, height: 48, borderRadius: 8, marginBottom: 8 }} />}
              <div className="card-row"><span>ID</span><span title={detail.id}>{String(detail.id).slice(0, 8)}…</span></div>
              <div className="card-row"><span>Name</span><span>{detail.name}</span></div>
              <div className="card-row"><span>Symbol</span><span>{detail.symbol}</span></div>
              <div className="card-row"><span>Chain ID</span><span>{detail.chain_id}</span></div>
              <div className="card-row"><span>Decimals</span><span>{detail.decimals}</span></div>
              <div className="card-row"><span>Contract</span><span title={detail.contract_address}>{detail.contract_address || '-'}</span></div>
              <div className="card-row"><span>Website</span><span>{detail.website || '-'}</span></div>
              <div className="card-row"><span>Status</span><span><span className={`badge ${detail.status === 'active' ? 'active' : ''}`}>{detail.status}</span></span></div>
              <div className="card-row"><span>Listing Type</span><span>{detail.listing_type}</span></div>
              <div className="card-row"><span>Created</span><span>{detail.created_at ? new Date(detail.created_at).toLocaleString() : '-'}</span></div>
              {detail.description && <p className="muted" style={{ marginTop: '0.5rem' }}>{detail.description}</p>}
            </div>
          </div>
        </section>
      )}

      {loading ? (
        <div className="state">Loading tokens…</div>
      ) : tokens.length === 0 ? (
        <div className="state">No tokens yet. Create one to get started.</div>
      ) : (
        <section>
          <div className="section-title"><h2>All Tokens ({tokens.length})</h2></div>
          <div className="coins-table">
            <table>
              <thead>
                <tr>
                  <th>Name</th><th>Symbol</th><th>Chain</th><th>Contract</th>
                  <th>Status</th><th>Type</th><th>Created</th><th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {tokens.map((t: any) => (
                  <tr key={t.id}>
                    <td>{t.logo_url && <img src={t.logo_url} alt="" />} {t.name}</td>
                    <td>{t.symbol}</td>
                    <td>{t.chain_id}</td>
                    <td title={t.contract_address}>{t.contract_address ? `${String(t.contract_address).slice(0, 10)}…` : '-'}</td>
                    <td><span className={`badge ${t.status === 'active' ? 'active' : ''}`}>{t.status}</span></td>
                    <td>{t.listing_type}</td>
                    <td>{t.created_at ? new Date(t.created_at).toLocaleDateString() : '-'}</td>
                    <td>
                      <div className="row-actions">
                        <button className="secondary" onClick={() => view(t.id)}>View</button>
                        <button className="secondary" onClick={() => startEdit(t)}>Edit</button>
                        <button className="danger" onClick={() => remove(t.id)}>Delete</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}
