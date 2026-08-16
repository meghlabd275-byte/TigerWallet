// API Keys Page - GET + POST /api/v1/api-keys
import React, { useState, useEffect } from 'react';
import { api, ApiKey } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function ApiKeys() {
  const { isDark } = useTheme();
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [busy, setBusy] = useState(false);
  const [revealId, setRevealId] = useState<string | null>(null);
  const [form, setForm] = useState({ exchange: 'binance', api_key: '' });

  const load = () => {
    setLoading(true);
    setError('');
    api.listApiKeys().then(data => {
      setKeys(data.api_keys || []);
      setLoading(false);
    }).catch(err => {
      setError(err.message || 'Failed to load API keys');
      setLoading(false);
    });
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await api.createApiKey({
        exchange: form.exchange,
        api_key: form.api_key,
      });
      setForm({ exchange: 'binance', api_key: '' });
      setShowCreate(false);
      load();
    } catch (err: any) {
      setError(err.message || 'Failed to create API key');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="resource-page">
      <header className="page-header">
        <h1>API Keys <span className="mode-pill">{isDark ? 'Dark' : 'Light'}</span></h1>
        <button className="btn-primary" onClick={() => setShowCreate(!showCreate)}>
          + New API Key
        </button>
      </header>

      <p className="hint">Keys are encrypted at rest (AES-256-GCM) on the WL-Bots backend.</p>

      {error && <div className="error">{error}</div>}

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          <div className="form-group">
            <label>Exchange</label>
            <input placeholder="binance" value={form.exchange}
              onChange={e => setForm({ ...form, exchange: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>API Key</label>
            <input placeholder="your-exchange-api-key" value={form.api_key}
              onChange={e => setForm({ ...form, api_key: e.target.value })} required />
          </div>
          <button type="submit" disabled={busy}>{busy ? 'Creating...' : 'Create'}</button>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : keys.length === 0 ? (
        <p>No API keys yet.</p>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
              <th>Exchange</th>
              <th>API Key</th>
              <th>Status</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {keys.map(k => (
              <tr key={k.id}>
                <td>{k.exchange}</td>
                <td className="api-key-cell">
                  {revealId === k.id ? k.api_key : (k.api_key_preview || '****')}
                  <button type="button" className="btn-link"
                    onClick={() => setRevealId(revealId === k.id ? null : k.id)}>
                    {revealId === k.id ? 'hide' : 'reveal'}
                  </button>
                </td>
                <td>
                  <span className={`status ${k.enabled ? 'running' : 'stopped'}`}>
                    {k.enabled ? 'enabled' : 'disabled'}
                  </span>
                </td>
                <td>{new Date(k.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
