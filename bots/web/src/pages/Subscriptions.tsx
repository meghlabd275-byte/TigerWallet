// Subscriptions Page - GET + POST /api/v1/subscriptions
import React, { useState, useEffect } from 'react';
import { api, Subscription } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

const TIERS = ['free', 'starter', 'pro', 'enterprise'];
const DURATIONS = [
  { label: '30 days', value: '720h' },
  { label: '90 days', value: '2160h' },
  { label: '1 year', value: '8760h' },
  { label: 'No expiry', value: '' },
];

export default function Subscriptions() {
  const { isDark } = useTheme();
  const [subs, setSubs] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({ tier: 'pro', expires_in: '720h' });

  const load = () => {
    setLoading(true);
    setError('');
    api.listSubscriptions().then(data => {
      setSubs(data.subscriptions || []);
      setLoading(false);
    }).catch(err => {
      setError(err.message || 'Failed to load subscriptions');
      setLoading(false);
    });
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await api.createSubscription({
        tier: form.tier,
        expires_in: form.expires_in || undefined,
      });
      setShowCreate(false);
      load();
    } catch (err: any) {
      setError(err.message || 'Failed to create subscription');
    } finally {
      setBusy(false);
    }
  };

  const fmtDate = (s: string | null) => (s ? new Date(s).toLocaleString() : 'Never');

  return (
    <div className="resource-page">
      <header className="page-header">
        <h1>Subscriptions <span className="mode-pill">{isDark ? 'Dark' : 'Light'}</span></h1>
        <button className="btn-primary" onClick={() => setShowCreate(!showCreate)}>
          + New Subscription
        </button>
      </header>

      {error && <div className="error">{error}</div>}

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          <div className="form-group">
            <label>Tier</label>
            <select value={form.tier} onChange={e => setForm({ ...form, tier: e.target.value })}>
              {TIERS.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Duration</label>
            <select value={form.expires_in} onChange={e => setForm({ ...form, expires_in: e.target.value })}>
              {DURATIONS.map(d => <option key={d.value || 'none'} value={d.value}>{d.label}</option>)}
            </select>
          </div>
          <button type="submit" disabled={busy}>{busy ? 'Creating...' : 'Create'}</button>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : subs.length === 0 ? (
        <p>No subscriptions yet.</p>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
              <th>Tier</th>
              <th>Started</th>
              <th>Expires</th>
            </tr>
          </thead>
          <tbody>
            {subs.map(s => (
              <tr key={s.id}>
                <td><span className="badge">{s.tier}</span></td>
                <td>{fmtDate(s.started_at)}</td>
                <td>{fmtDate(s.expires_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
