import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/project-party-api';

export default function Launchpad() {
  const [projects, setProjects] = useState([]);
  const [statusFilter, setStatusFilter] = useState('');
  const [selected, setSelected] = useState(null);
  const [participations, setParticipations] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [loading, setLoading] = useState(true);
  const [amount, setAmount] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    setLoading(true); setError(null);
    api.listLaunchpads(statusFilter || undefined)
      .then(d => setProjects((d && (d.launchpad_projects || d.projects)) || []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [statusFilter]);

  useEffect(load, [load]);

  const openProject = async (p) => {
    setSelected(p); setParticipations([]); setError(null); setNotice(null);
    try {
      const d = await api.listParticipations(p.id);
      setParticipations((d && (d.participations || d.items)) || []);
    } catch (_) {
      // a project with no participations yet returns an empty list
    }
  };

  const contribute = async (e) => {
    e.preventDefault();
    if (!selected) return;
    setBusy(true); setError(null); setNotice(null);
    try {
      const res = await api.contribute(selected.id, amount);
      setNotice(res && res.tx_hash
        ? `Contribution submitted to the blockchain network: ${res.tx_hash}`
        : 'Contribution submitted');
      setAmount('');
      openProject(selected);
    } catch (e2) {
      setError(e2.message);
    } finally {
      setBusy(false);
    }
  };

  const claim = async () => {
    if (!selected) return;
    setError(null); setNotice(null);
    try {
      const res = await api.claimTokens(selected.id);
      setNotice(res && res.tx_hash
        ? `Claim submitted to the blockchain network: ${res.tx_hash}`
        : 'Claim submitted');
      openProject(selected);
    } catch (e) {
      setError(e.message);
    }
  };

  return (
    <div className="launchpad-page">
      <h1>Launchpad</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <div className="filter-row">
        <label>Status
          <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}>
            {['', 'upcoming', 'active', 'ended', 'cancelled'].map(s => <option key={s} value={s}>{s || 'all'}</option>)}
          </select>
        </label>
      </div>

      {loading ? <p>Loading…</p> : projects.length === 0 ? (
        <p className="empty-state">No launchpad projects.</p>
      ) : (
        <table className="data-table">
          <thead><tr><th>Name</th><th>Status</th><th>Price</th><th>Supply</th><th>Actions</th></tr></thead>
          <tbody>
            {projects.map(p => (
              <tr key={p.id}>
                <td>{p.name || p.token_symbol || p.id}</td>
                <td>{p.status || '—'}</td>
                <td>{p.price_per_token != null ? p.price_per_token : '—'}</td>
                <td>{p.total_supply != null ? p.total_supply : '—'}</td>
                <td className="action-row"><button onClick={() => openProject(p)}>Open</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {selected && (
        <div className="detail-panel">
          <h2>{selected.name || selected.id}</h2>
          {selected.description && <p>{selected.description}</p>}
          <dl className="detail-list">
            <dt>Status</dt><dd>{selected.status || '—'}</dd>
            <dt>Price per token</dt><dd>{selected.price_per_token != null ? selected.price_per_token : '—'}</dd>
            <dt>Start</dt><dd>{selected.start_time || '—'}</dd>
            <dt>End</dt><dd>{selected.end_time || '—'}</dd>
          </dl>

          <form className="create-form" onSubmit={contribute}>
            <label>Contribution amount
              <input value={amount} onChange={e => setAmount(e.target.value)} required />
            </label>
            <button type="submit" disabled={busy}>{busy ? 'Submitting…' : 'Contribute on-chain'}</button>
            <button type="button" onClick={claim}>Claim tokens</button>
          </form>

          <h3>Participations</h3>
          {participations.length === 0 ? <p className="empty-state">No participations yet.</p> : (
            <table className="data-table">
              <thead><tr><th>User</th><th>Amount</th><th>Status</th><th>Tx</th></tr></thead>
              <tbody>
                {participations.map((x, i) => (
                  <tr key={x.id || i}>
                    <td>{x.user_id || x.wallet_address || '—'}</td>
                    <td>{x.amount != null ? x.amount : '—'}</td>
                    <td>{x.status || '—'}</td>
                    <td>{x.tx_hash || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}
