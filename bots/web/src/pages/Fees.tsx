// Fees Page - GET + POST /api/v1/fees
import React, { useState, useEffect } from 'react';
import { api, FeeConfig } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Fees() {
  const { isDark } = useTheme();
  const [fees, setFees] = useState<FeeConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({ name: '', percentage: '0.10', enabled: true });

  const load = () => {
    setLoading(true);
    setError('');
    api.listFeeConfigs().then(data => {
      setFees(data.fee_configs || []);
      setLoading(false);
    }).catch(err => {
      setError(err.message || 'Failed to load fee configs');
      setLoading(false);
    });
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try {
      await api.createFeeConfig({
        name: form.name,
        percentage: form.percentage,
        enabled: form.enabled,
      });
      setForm({ name: '', percentage: '0.10', enabled: true });
      setShowCreate(false);
      load();
    } catch (err: any) {
      setError(err.message || 'Failed to create fee config');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="resource-page">
      <header className="page-header">
        <h1>Fee Configs <span className="mode-pill">{isDark ? 'Dark' : 'Light'}</span></h1>
        <button className="btn-primary" onClick={() => setShowCreate(!showCreate)}>
          + New Fee Config
        </button>
      </header>

      {error && <div className="error">{error}</div>}

      {showCreate && (
        <form className="create-form" onSubmit={handleCreate}>
          <div className="form-group">
            <label>Name</label>
            <input placeholder="e.g. Taker Fee" value={form.name}
              onChange={e => setForm({ ...form, name: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>Percentage</label>
            <input type="number" step="0.001" placeholder="0.10" value={form.percentage}
              onChange={e => setForm({ ...form, percentage: e.target.value })} required />
          </div>
          <div className="form-group">
            <label>
              <input type="checkbox" style={{ width: 'auto' }}
                checked={form.enabled}
                onChange={e => setForm({ ...form, enabled: e.target.checked })} />
              {' '}Enabled
            </label>
          </div>
          <button type="submit" disabled={busy}>{busy ? 'Creating...' : 'Create'}</button>
        </form>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : fees.length === 0 ? (
        <p>No fee configs yet.</p>
      ) : (
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Percentage</th>
              <th>Enabled</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {fees.map(f => (
              <tr key={f.id}>
                <td>{f.name}</td>
                <td>{f.percentage}%</td>
                <td>
                  <span className={`status ${f.enabled ? 'running' : 'stopped'}`}>
                    {f.enabled ? 'enabled' : 'disabled'}
                  </span>
                </td>
                <td>{new Date(f.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
