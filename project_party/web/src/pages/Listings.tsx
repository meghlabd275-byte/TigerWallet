// Listings Page — WL-ProjectParty. Real backend coverage:
// POST /listings (create), GET /listings (list).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const LAUNCH_TYPES = ['fair_launch', 'presale', 'farming'];
const PAIR_TOKENS = ['USDT', 'USDC', 'ETH', 'BNB'];

interface LForm {
  token_id: string; pair_token: string; initial_price: string;
  launch_type: string; start_time: string; end_time: string;
}
const EMPTY: LForm = {
  token_id: '', pair_token: 'USDT', initial_price: '',
  launch_type: 'fair_launch', start_time: '', end_time: ''
};

function toRFC3339(local: string) {
  if (!local) return undefined;
  const d = new Date(local);
  if (isNaN(d.getTime())) return undefined;
  return d.toISOString();
}

export default function Listings() {
  const [listings, setListings] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState<LForm>(EMPTY);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.listListings();
      setListings(data.listings || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load listings');
    }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.createListing({
        token_id: form.token_id,
        pair_token: form.pair_token,
        initial_price: form.initial_price || undefined,
        launch_type: form.launch_type,
        start_time: toRFC3339(form.start_time),
        end_time: toRFC3339(form.end_time)
      });
      setMsg({ type: 'success', text: 'Listing created.' });
      setForm(EMPTY);
      setShowForm(false);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to create listing' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Token Listings</h1>
        <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Create Listing'}</button>
      </div>
      <p className="subtitle">Token launches (fair launch / presale / farming) paired against a quote token.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>Create Listing</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} placeholder="uuid" required />
              </div>
              <div className="form-field">
                <label>Pair Token</label>
                <select value={form.pair_token} onChange={e => setForm({ ...form, pair_token: e.target.value })}>
                  {PAIR_TOKENS.map(p => <option key={p} value={p}>{p}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Initial Price</label>
                <input value={form.initial_price} onChange={e => setForm({ ...form, initial_price: e.target.value })} placeholder="0.00" />
              </div>
              <div className="form-field">
                <label>Launch Type</label>
                <select value={form.launch_type} onChange={e => setForm({ ...form, launch_type: e.target.value })}>
                  {LAUNCH_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Start Time</label>
                <input type="datetime-local" value={form.start_time} onChange={e => setForm({ ...form, start_time: e.target.value })} />
              </div>
              <div className="form-field">
                <label>End Time</label>
                <input type="datetime-local" value={form.end_time} onChange={e => setForm({ ...form, end_time: e.target.value })} />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Creating…' : 'Create'}</button>
              <button type="button" className="secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {loading ? (
        <div className="state">Loading listings…</div>
      ) : listings.length === 0 ? (
        <div className="state">No listings yet.</div>
      ) : (
        <section>
          <div className="section-title"><h2>All Listings ({listings.length})</h2></div>
          <div className="coins-table">
            <table>
              <thead>
                <tr>
                  <th>Token</th><th>Pair</th><th>Type</th><th>Initial</th>
                  <th>Current</th><th>24h Vol</th><th>Liquidity</th>
                  <th>Mkt Cap</th><th>24h %</th><th>Status</th><th>Window</th>
                </tr>
              </thead>
              <tbody>
                {listings.map((l: any) => (
                  <tr key={l.id}>
                    <td title={l.token_id}>{String(l.token_id).slice(0, 8)}…</td>
                    <td>{l.pair_token}</td>
                    <td>{l.launch_type}</td>
                    <td>{l.initial_price ?? '-'}</td>
                    <td>{l.current_price ?? '-'}</td>
                    <td>{l.volume_24h ?? '-'}</td>
                    <td>{l.liquidity_usd ?? '-'}</td>
                    <td>{l.market_cap ?? '-'}</td>
                    <td className={l.price_change_24h >= 0 ? 'up' : 'down'}>{l.price_change_24h ?? '-'}</td>
                    <td><span className={`badge ${l.status === 'active' ? 'active' : ''}`}>{l.status}</span></td>
                    <td>{l.start_time || l.end_time ? `${l.start_time ? new Date(l.start_time).toLocaleDateString() : '—'} → ${l.end_time ? new Date(l.end_time).toLocaleDateString() : '—'}` : '-'}</td>
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
