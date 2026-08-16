// Favorites Page — WL-ProjectParty. Real backend coverage:
// POST /favorites (add), GET /favorites (list), DELETE /favorites/:id (remove).
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

interface FavForm { token_id: string; notes: string; }
const EMPTY: FavForm = { token_id: '', notes: '' };

export default function Favorites() {
  const [favs, setFavs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<FavForm>(EMPTY);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await api.listFavorites();
      setFavs(data.favorites || []);
    } catch (e: any) {
      setError(e.message || 'Failed to load favorites');
    }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMsg(null);
    try {
      await api.addFavorite({ token_id: form.token_id, notes: form.notes || undefined });
      setMsg({ type: 'success', text: 'Favorite added.' });
      setForm(EMPTY);
      setShowForm(false);
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to add favorite' });
    }
    setSubmitting(false);
  };

  const remove = async (id: string) => {
    if (!window.confirm('Remove this favorite?')) return;
    try {
      await api.removeFavorite(id);
      setMsg({ type: 'success', text: 'Favorite removed.' });
      load();
    } catch (err: any) {
      setMsg({ type: 'error', text: err.message || 'Failed to remove favorite' });
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <h1>Favorites</h1>
        <button onClick={() => setShowForm(s => !s)}>{showForm ? 'Close' : 'Add Favorite'}</button>
      </div>
      <p className="subtitle">Track favorite tokens with optional notes. Remove anytime.</p>

      {msg && <div className={`alert ${msg.type}`}>{msg.text}</div>}
      {error && <div className="alert error">{error}</div>}

      {showForm && (
        <section>
          <div className="section-title"><h2>Add Favorite</h2></div>
          <form onSubmit={submit}>
            <div className="form-grid">
              <div className="form-field">
                <label>Token ID (UUID)</label>
                <input value={form.token_id} onChange={e => setForm({ ...form, token_id: e.target.value })} required />
              </div>
              <div className="form-field" style={{ gridColumn: '1 / -1' }}>
                <label>Notes (optional)</label>
                <textarea value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
            </div>
            <div className="form-actions">
              <button type="submit" disabled={submitting}>{submitting ? 'Adding…' : 'Add'}</button>
              <button type="button" className="secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </form>
        </section>
      )}

      {loading ? (
        <div className="state">Loading favorites…</div>
      ) : favs.length === 0 ? (
        <div className="state">No favorites yet.</div>
      ) : (
        <div className="cards-grid">
          {favs.map((f: any) => (
            <div className="card" key={f.id}>
              <div className="card-row"><span>Token</span><span title={f.token_id}>{String(f.token_id).slice(0, 8)}…</span></div>
              <div className="card-row"><span>Added</span><span>{f.created_at ? new Date(f.created_at).toLocaleString() : '-'}</span></div>
              {f.notes && <p className="muted" style={{ marginTop: '0.4rem' }}>{f.notes}</p>}
              <div className="row-actions" style={{ marginTop: '0.6rem' }}>
                <button className="danger" onClick={() => remove(f.id)}>Remove</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
