// Fees Page — WL-ProjectParty. Real backend coverage:
// POST /fees (create config), GET /fees (list configs).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

const FEE_TYPES = ['listing', 'trading', 'withdrawal', 'participation'];

interface FeeForm {
  token_id: string; fee_type: string; fee_percentage: string;
  min_fee: string; max_fee: string;
}

const EMPTY: FeeForm = { token_id: '', fee_type: 'listing', fee_percentage: '', min_fee: '', max_fee: '' };

export default function Fees() {
  const [configs, setConfigs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<FeeForm>(EMPTY);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.listFeeConfigs();
      setConfigs(data.fee_configs || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load fee configs');
    }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.createFeeConfig({
        token_id: form.token_id,
        fee_type: form.fee_type,
        fee_percentage: form.fee_percentage || undefined,
        min_fee: form.min_fee || undefined,
        max_fee: form.max_fee || undefined
      });
      setMsg({ type: 'success', text: 'Fee config created.' });
      setForm(EMPTY);
      setShowForm(false);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to create fee config' });
    }
    setSubmitting(false);
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Fees</h1>
        <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Create Fee Config'}</button>
      </div>
      <p className="subtitle">Create and view fee configs (listing, trading, withdrawal, participation).</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>Create Fee Config</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} required />
              </div>
              <div className="form-field">
                <label>Fee Type</label>
                <select value={form.fee_type} onChange={e => setForm({ ...form, fee_type: e.target.value })}>
                  {FEE_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div className="form-field">
                <label>Fee Percentage</label>
                <input value={form.fee_percentage} onChange={e => setForm({ ...form, fee_percentage: e.target.value })} placeholder="e.g. 1.5" />
              </div>
              <div className="form-field">
                <label>Min Fee</label>
                <input value={form.min_fee} onChange={e => setForm({ ...form, min_fee: e.target.value })} placeholder="e.g. 0.0" />
              </div>
              <div className="form-field">
                <label>Max Fee</label>
                <input value={form.max_fee} onChange={e => setForm({ ...form, max_fee: e.target.value })} placeholder="e.g. 1000.0" />
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
        <div className="state">Loading fee configs…</div>
      ) : configs.length === 0 ? (
        <div className="state">No fee configs yet.</div>
      ) : (
        <section>
          <div className="section-title"><h2>Configs ({configs.length})</h2></div>
          <div className="coins-table">
            <table>
              <thead>
                <tr><th>ID</th><th>Token</th><th>Type</th><th>Percentage</th><th>Min</th><th>Max</th><th>Created</th></tr>
              </thead>
              <tbody>
                {configs.map((f: any) => (
                  <tr key={f.id}>
                    <td title={f.id}>{String(f.id).slice(0, 8)}…</td>
                    <td title={f.token_id}>{String(f.token_id).slice(0, 8)}…</td>
                    <td>{f.fee_type}</td>
                    <td>{f.fee_percentage || '-'}</td>
                    <td>{f.min_fee || '-'}</td>
                    <td>{f.max_fee || '-'}</td>
                    <td>{f.created_at ? new Date(f.created_at).toLocaleString() : '-'}</td>
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
