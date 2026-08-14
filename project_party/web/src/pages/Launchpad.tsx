// Launchpad Page - ProjectParty (IDO / Presale)
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const PAYMENTS = ['USDT', 'USDC', 'BNB', 'ETH'];

export default function Launchpad() {
  const [launchpads, setLaunchpads] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form, setForm] = useState({
    token_id: '', name: '', description: '', soft_cap: '', hard_cap: '',
    min_contribution: '', max_contribution: '', start_time: '', end_time: '',
    token_price: '', accepted_payment: 'USDT'
  });
  const [actionMsg, setActionMsg] = useState<{ id: string; type: 'success' | 'error'; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.getLaunchpads();
      setLaunchpads(data.launchpads || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load launchpads');
    }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setActionMsg(null);
    try {
      await api.createLaunchpad({
        token_id: form.token_id,
        name: form.name,
        description: form.description,
        soft_cap: form.soft_cap,
        hard_cap: form.hard_cap,
        min_contribution: form.min_contribution,
        max_contribution: form.max_contribution,
        start_time: new Date(form.start_time).toISOString(),
        end_time: new Date(form.end_time).toISOString(),
        token_price: form.token_price,
        accepted_payment: form.accepted_payment
      });
      setActionMsg({ id: '', type: 'success', text: 'Launchpad campaign created.' });
      setForm({ token_id: '', name: '', description: '', soft_cap: '', hard_cap: '', min_contribution: '', max_contribution: '', start_time: '', end_time: '', token_price: '', accepted_payment: 'USDT' });
      setShowForm(false);
      load();
    } catch (err: any) {
      setActionMsg({ id: '', type: 'error', text: err.message || 'Failed to create launchpad' });
    }
    setSubmitting(false);
  };

  const contribute = async (id: string, amount: string, userId: string) => {
    try {
      await api.contributeLaunchpad(id, amount, userId || undefined);
      setActionMsg({ id, type: 'success', text: 'Contribution submitted.' });
      load();
    } catch (err: any) {
      setActionMsg({ id, type: 'error', text: err.message || 'Contribution failed' });
    }
  };

  const claim = async (id: string, userId: string) => {
    if (!userId) { setActionMsg({ id, type: 'error', text: 'Enter a user ID to claim.' }); return; }
    try {
      await api.claimLaunchpad(id, userId);
      setActionMsg({ id, type: 'success', text: 'Tokens claimed successfully.' });
    } catch (err: any) {
      setActionMsg({ id, type: 'error', text: err.message || 'Claim failed' });
    }
  };

  const cancel = async (id: string) => {
    try {
      await api.cancelLaunchpad(id);
      setActionMsg({ id, type: 'success', text: 'Launchpad cancelled.' });
      load();
    } catch (err: any) {
      setActionMsg({ id, type: 'error', text: err.message || 'Cancel failed' });
    }
  };

  const progress = (raised: string, hard: string) => {
    const r = parseFloat(raised) || 0;
    const h = parseFloat(hard) || 0;
    return h > 0 ? Math.min(100, (r / h) * 100) : 0;
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Launchpad (IDO / Presale)</h1>
        <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Create Campaign'}</button>
      </div>
      <p className="subtitle">Browse and participate in token presales with soft/hard caps.</p>

      {actionMsg && !actionMsg.id && <div className={`alert ${actionMsg.type}`}>{actionMsg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>Create Launchpad Campaign</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Campaign Name</label>
                <input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Soft Cap</label>
                <input value={form.soft_cap} onChange={e => setForm({ ...form, soft_cap: e.target.value })} placeholder="e.g. 10000" required />
              </div>
              <div className="form-field">
                <label>Hard Cap</label>
                <input value={form.hard_cap} onChange={e => setForm({ ...form, hard_cap: e.target.value })} placeholder="e.g. 50000" required />
              </div>
              <div className="form-field">
                <label>Min Contribution</label>
                <input value={form.min_contribution} onChange={e => setForm({ ...form, min_contribution: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Max Contribution</label>
                <input value={form.max_contribution} onChange={e => setForm({ ...form, max_contribution: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Token Price</label>
                <input value={form.token_price} onChange={e => setForm({ ...form, token_price: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Accepted Payment</label>
                <select value={form.accepted_payment} onChange={e => setForm({ ...form, accepted_payment: e.target.value })}>
                  {PAYMENTS.map(p => <option key={p} value={p}>{p}</option>)}
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
              <div className="form-field" style={{ gridColumn: '1 / -1' }}>
                <label>Description</label>
                <textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} />
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
        <div className="state">Loading launchpads...</div>
      ) : launchpads.length === 0 ? (
        <div className="state">No data available</div>
      ) : (
        <div className="cards-grid">
          {launchpads.map((lp: any) => (
            <LaunchpadCard
              key={lp.id}
              lp={lp}
              progress={progress}
              actionMsg={actionMsg?.id === lp.id ? actionMsg : null}
              onContribute={contribute}
              onClaim={claim}
              onCancel={cancel}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function LaunchpadCard({
  lp, progress, actionMsg, onContribute, onClaim, onCancel
}: any) {
  const [amount, setAmount] = useState('');
  const [userId, setUserId] = useState('');

  return (
    <div className="card">
      <div className="card-row"><span>Campaign</span><span><strong>{lp.name}</strong></span></div>
      <div className="card-row"><span>Symbol</span><span>{lp.symbol || '-'}</span></div>
      <div className="card-row"><span>Status</span><span><span className={`badge ${lp.status === 'active' ? 'active' : ''}`}>{lp.status}</span></span></div>
      <div className="card-row"><span>Soft / Hard Cap</span><span>{lp.soft_cap} / {lp.hard_cap}</span></div>
      <div className="card-row"><span>Token Price</span><span>{lp.token_price} ({lp.accepted_payment})</span></div>
      <div className="card-row"><span>Min / Max</span><span>{lp.min_contribution} / {lp.max_contribution}</span></div>
      <div className="card-row"><span>Total Raised</span><span>{lp.total_raised}</span></div>
      <div className="card-row"><span>Contributors</span><span>{lp.contributors ?? 0}</span></div>

      <div style={{ marginTop: '0.6rem' }}>
        <div className="muted" style={{ fontSize: '0.8rem' }}>Progress to hard cap</div>
        <div className="progress"><div style={{ width: `${progress(lp.total_raised, lp.hard_cap)}%` }} /></div>
      </div>

      <div className="muted" style={{ fontSize: '0.78rem', marginTop: '0.4rem' }}>
        {new Date(lp.start_time).toLocaleString()} - {new Date(lp.end_time).toLocaleString()}
      </div>

      {actionMsg && <div className={`alert ${actionMsg.type}`} style={{ marginTop: '0.6rem', marginBottom: 0 }}>{actionMsg.text}</div>}

      <div style={{ marginTop: '0.8rem', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
        <input value={userId} onChange={e => setUserId(e.target.value)} placeholder="Your user ID (optional)" />
        <input value={amount} onChange={e => setAmount(e.target.value)} placeholder="Contribution amount" />
        <div className="row-actions">
          <button onClick={() => onContribute(lp.id, amount, userId)}>Contribute</button>
          <button className="secondary" onClick={() => onClaim(lp.id, userId)}>Claim</button>
          <button className="danger" onClick={() => onCancel(lp.id)}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
