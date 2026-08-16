// Submit Token Page — WL-ProjectParty. Real backend: POST /tokens (createToken).
// This is a lightweight token-submission form; full CRUD lives on the Tokens page.
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../services/api';

const CHAINS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' }
];

export default function Submit() {
  const [form, setForm] = useState({
    name: '', symbol: '', contract_address: '', chain_id: 1,
    decimals: 18, logo_url: '', website: '', description: '',
    status: 'pending', listing_type: 'standard'
  });
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.createToken({
        name: form.name,
        symbol: form.symbol,
        contract_address: form.contract_address || undefined,
        chain_id: form.chain_id,
        decimals: form.decimals,
        logo_url: form.logo_url || undefined,
        website: form.website || undefined,
        description: form.description || undefined,
        status: form.status,
        listing_type: form.listing_type
      });
      setMsg({ type: 'success', text: 'Token submitted! It is now pending review.' });
      setForm({
        name: '', symbol: '', contract_address: '', chain_id: 1,
        decimals: 18, logo_url: '', website: '', description: '',
        status: 'pending', listing_type: 'standard'
      });
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to submit token' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page submit-page">
      <div className="page-header">
        <h1>Submit Token</h1>
        <button className="secondary" onClick={() => navigate('/tokens')}>View all tokens →</button>
      </div>
      <p className="subtitle">Submit a new token to the WL-ProjectParty backend. Defaults to “pending” status.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}

      <form onSubmit={handleSubmit}>
        <div className="form-grid">
          <div className="form-field">
            <label>Token Name</label>
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
            <label>Listing Type</label>
            <select value={form.listing_type} onChange={e => setForm({ ...form, listing_type: e.target.value })}>
              <option value="standard">standard</option>
              <option value="premium">premium</option>
              <option value="featured">featured</option>
            </select>
          </div>
          <div className="form-field" style={{ gridColumn: '1 / -1' }}>
            <label>Description</label>
            <textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
          </div>
        </div>
        <div className="form-actions">
          <button type="submit" disabled={submitting}>{submitting ? 'Submitting…' : 'Submit Token'}</button>
        </div>
      </form>
    </div>
  );
}
