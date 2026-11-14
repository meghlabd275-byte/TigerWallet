import React, { useEffect, useState, useCallback } from 'react';
import api from '../services/bot-api';

const TABS = [
  { id: 'cex', label: 'CEX' },
  { id: 'dex', label: 'DEX' },
  { id: 'keys', label: 'API Keys' },
];

export default function Connectors() {
  const [tab, setTab] = useState('cex');
  const [cex, setCex] = useState([]);
  const [dex, setDex] = useState([]);
  const [keys, setKeys] = useState([]);
  const [error, setError] = useState(null);
  const [notice, setNotice] = useState(null);
  const [form, setForm] = useState({ name: '', apiKey: '', exchange: '' });

  const load = useCallback(() => {
    setError(null);
    Promise.all([
      api.listCEX().catch(() => ({})),
      api.listDEX().catch(() => ({})),
      api.listAPIKeys().catch(() => ({})),
    ]).then(([c, d, k]) => {
      setCex((c && (c.connectors || c.cex)) || []);
      setDex((d && (d.connectors || d.dex)) || []);
      setKeys((k && (k.keys || k.api_keys)) || []);
    }).catch(e => setError(e.message));
  }, []);

  useEffect(load, [load]);

  const run = async (fn, msg) => {
    setError(null); setNotice(null);
    try { await fn(); setNotice(msg); load(); }
    catch (e) { setError(e.message); }
  };

  const renderList = (items, removeFn, cols) => (
    items.length === 0 ? <p className="empty-state">None configured.</p> : (
      <table className="data-table">
        <thead><tr>{cols.map(c => <th key={c.key}>{c.label}</th>)}<th>Actions</th></tr></thead>
        <tbody>
          {items.map((it, i) => (
            <tr key={it.id || i}>
              {cols.map(c => <td key={c.key}>{it[c.key] || '—'}</td>)}
              <td className="action-row">
                <button onClick={() => run(() => removeFn(it.id), 'Removed')}>Remove</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    )
  );

  return (
    <div className="connectors-page">
      <h1>Connectors</h1>
      {error && <p className="error-banner">{error}</p>}
      {notice && <p className="success-banner">{notice}</p>}

      <div className="tabs">
        {TABS.map(t => (
          <button key={t.id} className={tab === t.id ? 'active' : ''} onClick={() => setTab(t.id)}>{t.label}</button>
        ))}
      </div>

      {tab === 'cex' && (
        <>
          <form className="create-form" onSubmit={(e) => { e.preventDefault(); run(() => api.addCEX({ name: form.name, config: {} }), 'CEX connector added'); setForm({ ...form, name: '' }); }}>
            <label>Name<input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required /></label>
            <button type="submit">Add CEX</button>
          </form>
          {renderList(cex, api.removeCEX, [{ key: 'name', label: 'Name' }, { key: 'status', label: 'Status' }])}
        </>
      )}

      {tab === 'dex' && (
        <>
          <form className="create-form" onSubmit={(e) => { e.preventDefault(); run(() => api.addDEX({ name: form.name, config: {} }), 'DEX connector added'); setForm({ ...form, name: '' }); }}>
            <label>Name<input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required /></label>
            <button type="submit">Add DEX</button>
          </form>
          {renderList(dex, api.removeDEX, [{ key: 'name', label: 'Name' }, { key: 'chain', label: 'Chain' }])}
        </>
      )}

      {tab === 'keys' && (
        <>
          <form className="create-form" onSubmit={(e) => { e.preventDefault(); run(() => api.createAPIKey({ exchange: form.exchange, api_key: form.apiKey }), 'API key stored'); setForm({ ...form, exchange: '', apiKey: '' }); }}>
            <label>Exchange<input value={form.exchange} onChange={e => setForm({ ...form, exchange: e.target.value })} required /></label>
            <label>API key<input type="password" value={form.apiKey} onChange={e => setForm({ ...form, apiKey: e.target.value })} required /></label>
            <button type="submit">Add key</button>
          </form>
          {renderList(keys, api.deleteAPIKey, [{ key: 'exchange', label: 'Exchange' }, { key: 'created_at', label: 'Created' }])}
        </>
      )}
    </div>
  );
}
