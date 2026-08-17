import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const DEVICE_TYPES = [
  { value: 'desktop', label: 'Desktop' },
  { value: 'mobile', label: 'Mobile' },
  { value: 'tablet', label: 'Tablet' },
  { value: 'extension', label: 'Browser Extension' },
];

function Devices() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');

  const [form, setForm] = useState({ name: '', deviceType: 'desktop' });
  const [busy, setBusy] = useState(false);
  const [actionId, setActionId] = useState(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.getDevices()
      .then((data) => { if (alive) { setDevices(data.devices || []); setLoading(false); } })
      .catch((err) => { if (alive) { setError(err.message || 'Failed to load devices'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const reload = async () => {
    const data = await api.getDevices();
    setDevices(data.devices || []);
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    if (!form.name.trim()) { setError('Device name is required'); return; }
    setBusy(true);
    try {
      await api.registerDevice({ name: form.name.trim(), deviceType: form.deviceType });
      setForm({ name: '', deviceType: 'desktop' });
      await reload();
      setInfo('Device registered.');
    } catch (err) {
      setError(err.message || 'Failed to register device');
    } finally {
      setBusy(false);
    }
  };

  const handleSync = async (d) => {
    setError('');
    setInfo('');
    const id = d.id || d.device_id || d.deviceId;
    setActionId(id);
    try {
      await api.syncDevice(id);
      setInfo(`Synced ${d.name || 'device'}.`);
      await reload();
    } catch (err) {
      setError(err.message || 'Failed to sync device');
    } finally {
      setActionId(null);
    }
  };

  const handleDelete = async (d) => {
    setError('');
    setInfo('');
    if (!window.confirm(`Delete device "${d.name}"?`)) return;
    const id = d.id || d.device_id || d.deviceId;
    setActionId(id);
    try {
      await api.deleteDevice(id);
      setInfo('Device deleted.');
      await reload();
    } catch (err) {
      setError(err.message || 'Failed to delete device');
    } finally {
      setActionId(null);
    }
  };

  return (
    <div className="wallets-page">
      <header className="page-header">
        <h1>Devices</h1>
      </header>

      {error && <div className="error">{error}</div>}
      {info && <div className="success-banner" style={{ marginBottom: '16px' }}><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}

      <form className="import-form" style={{ maxWidth: '600px' }} onSubmit={handleRegister}>
        <label style={{ fontSize: '0.85rem', color: 'var(--text-secondary)' }}>Register device</label>
        <input placeholder="Device name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
        <select value={form.deviceType} onChange={(e) => setForm({ ...form, deviceType: e.target.value })}>
          {DEVICE_TYPES.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select>
        <div className="mnemonic-actions">
          <button type="submit" disabled={busy}>{busy ? 'Registering…' : 'Register'}</button>
        </div>
      </form>

      {loading ? (
        <p>Loading...</p>
      ) : devices.length === 0 ? (
        <p>No devices registered yet.</p>
      ) : (
        <div className="wallets-grid" style={{ marginTop: '20px' }}>
          {devices.map((d, idx) => {
            const id = d.id || d.device_id || d.deviceId || idx;
            return (
              <div key={id} className="wallet-card">
                <h3>{d.name || 'Unnamed device'}</h3>
                <p className="network">{d.device_type || d.deviceType || 'device'}</p>
                {d.last_synced && <p className="address">Last synced: {d.last_synced}</p>}
                <div className="mnemonic-actions" style={{ marginTop: '12px' }}>
                  <button onClick={() => handleSync(d)} disabled={actionId === id}>
                    {actionId === id ? 'Syncing…' : '🔄 Sync'}
                  </button>
                  <button onClick={() => handleDelete(d)} disabled={actionId === id}>🗑️ Delete</button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default Devices;
