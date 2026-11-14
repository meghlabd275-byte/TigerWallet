import React, { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import api from '../services/project-party-api';

const STATUSES = ['', 'draft', 'submitted', 'in_review', 'listed', 'rejected'];

export default function Tokens() {
  const [tokens, setTokens] = useState([]);
  const [statusFilter, setStatusFilter] = useState('');
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({
    name: '', symbol: '', chain: 'ethereum', contract_address: '',
    decimals: '18', total_supply: '', description: ''
  });
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    api.listTokens(statusFilter || undefined)
      .then(d => setTokens((d && d.tokens) || []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [statusFilter]);

  useEffect(load, [load]);

  const create = async (e) => {
    e.preventDefault();
    setBusy(true); setError(null); setNotice(null);
    try {
      await api.createToken({
        name: form.name,
        symbol: form.symbol,
        chain: form.chain,
        contract_address: form.contract_address || undefined,
        decimals: Number(form.decimals) || 18,
        total_supply: form.total_supply,
        description: form.description || undefined,
      });
      setNotice('Token submitted (status is set by the backend workflow)');
      setForm({ name: '', symbol: '', chain: 'ethereum', contract_address: '', decimals: '18', total_supply: '', description: '' });
      load();
    } catch (e2) {
      setError(e2.message);
    } finally {
      setBusy(false);
    }
  };

  const submitForReview = async (id) => {
    setError(null); setNotice(null);
    try { await api.submitToken(id); setNotice('Token submitted for review'); load(); }
    catch (e) { setError(e.message); }
  };

  return (
    <div className="tokens-page">
      <h1>Tokens</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <form className="create-form" onSubmit={create}>
        <h2>Register a token</h2>
        <label>Name<input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required /></label>
        <label>Symbol<input value={form.symbol} onChange={e => setForm({ ...form, symbol: e.target.value })} required /></label>
        <label>Chain
          <select value={form.chain} onChange={e => setForm({ ...form, chain: e.target.value })}>
            {['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'avalanche', 'base', 'tigerchain'].map(c => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
        </label>
        <label>Contract address<input value={form.contract_address} onChange={e => setForm({ ...form, contract_address: e.target.value })} /></label>
        <label>Decimals<input value={form.decimals} onChange={e => setForm({ ...form, decimals: e.target.value })} /></label>
        <label>Total supply<input value={form.total_supply} onChange={e => setForm({ ...form, total_supply: e.target.value })} /></label>
        <label>Description<input value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} /></label>
        <button type="submit" disabled={busy}>{busy ? 'Submitting…' : 'Register token'}</button>
      </form>

      <h2>Your tokens</h2>
      <div className="filter-row">
        <label>Status
          <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}>
            {STATUSES.map(s => <option key={s} value={s}>{s || 'all'}</option>)}
          </select>
        </label>
      </div>

      {loading ? <p>Loading…</p> : tokens.length === 0 ? (
        <p className="empty-state">No tokens registered.</p>
      ) : (
        <table className="data-table">
          <thead><tr><th>Symbol</th><th>Name</th><th>Chain</th><th>Status</th><th>Verified</th><th>Actions</th></tr></thead>
          <tbody>
            {tokens.map(t => (
              <tr key={t.id}>
                <td><Link to={`/tokens/${t.id}`}>{t.symbol}</Link></td>
                <td>{t.name}</td>
                <td>{t.chain || '—'}</td>
                <td>{t.status || '—'}</td>
                <td>{t.contract_verified ? 'yes' : 'no'}</td>
                <td className="action-row">
                  <button onClick={() => submitForReview(t.id)}>Submit for review</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
