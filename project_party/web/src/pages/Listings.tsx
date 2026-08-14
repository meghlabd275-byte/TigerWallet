// Listings Page - ProjectParty
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const STATUSES = ['upcoming', 'active', 'completed', 'cancelled'];
const LAUNCH_TYPES = ['fair_launch', 'presale', 'farming'];

export default function Listings() {
  const [listings, setListings] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState({
    token_id: '', pair_token: 'USDT', initial_price: '',
    launch_type: 'fair_launch', start_time: '', end_time: ''
  });
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.getListings();
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
      const startISO = new Date(form.start_time).toISOString();
      const endISO = new Date(form.end_time).toISOString();
      await api.createListing({
        token_id: form.token_id,
        pair_token: form.pair_token,
        initial_price: form.initial_price,
        launch_type: form.launch_type,
        start_time: startISO,
        end_time: endISO
      });
      setMsg({ type: 'success', text: 'Listing created.' });
      setForm({ token_id: '', pair_token: 'USDT', initial_price: '', launch_type: 'fair_launch', start_time: '', end_time: '' });
      setShowForm(false);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to create listing' });
    }
    setSubmitting(false);
  };

  const changeStatus = async (id: string, status: string) => {
    try {
      await api.updateListingStatus(id, status);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to update status' });
    }
  };

  const feature = async (id: string) => {
    try {
      await api.featureListing(id);
      setMsg({ type: 'success', text: 'Listing featured.' });
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to feature listing' });
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Token Listings</h1>
        <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Create Listing'}</button>
      </div>
      <p className="subtitle">Token launches with pricing, volume, liquidity and market-cap metrics.</p>

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
                  <option value="USDT">USDT</option>
                  <option value="USDC">USDC</option>
                  <option value="ETH">ETH</option>
                  <option value="BNB">BNB</option>
                </select>
              </div>
              <div className="form-field">
                <label>Initial Price</label>
                <input value={form.initial_price} onChange={e => setForm({ ...form, initial_price: e.target.value })} placeholder="0.00" required />
              </div>
              <div className="form-field">
                <label>Launch Type</label>
                <select value={form.launch_type} onChange={e => setForm({ ...form, launch_type: e.target.value })}>
                  {LAUNCH_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Start Time</label>
                <input type="datetime-local" value={form.start_time} onChange={e => setForm({ ...form, start_time: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>End Time</label>
                <input type="datetime-local" value={form.end_time} onChange={e => setForm({ ...form, end_time: e.target.value })} required />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Creating...' : 'Create'}</button>
              <button type="button" className="secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {loading ? (
        <div className="state">Loading listings...</div>
      ) : listings.length === 0 ? (
        <div className="state">No data available</div>
      ) : (
        <section>
          <div className="section-title"><h2>All Listings ({listings.length})</h2></div>
          <table>
            <thead>
              <tr>
                <th>Token</th>
                <th>Pair</th>
                <th>Type</th>
                <th>Initial</th>
                <th>Current</th>
                <th>24h Vol</th>
                <th>Liquidity</th>
                <th>Mkt Cap</th>
                <th>24h %</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {listings.map((l: any) => (
                <tr key={l.id}>
                  <td title={l.token_id}>{String(l.token_id).slice(0, 8)}...</td>
                  <td>{l.pair_token}</td>
                  <td>{l.launch_type}</td>
                  <td>${l.initial_price}</td>
                  <td>${l.current_price}</td>
                  <td>${l.volume_24h}</td>
                  <td>${l.liquidity_usd}</td>
                  <td>${l.market_cap}</td>
                  <td className={l.price_change_24h >= 0 ? 'up' : 'down'}>{l.price_change_24h}%</td>
                  <td><span className={`badge ${l.status === 'active' ? 'active' : ''}`}>{l.status}</span></td>
                  <td>
                    <div className="row-actions">
                      <select value={l.status} onChange={e => changeStatus(l.id, e.target.value)}>
                        {STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
                      </select>
                      <button className="secondary" onClick={() => feature(l.id)}>Feature</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}
    </div>
  );
}
