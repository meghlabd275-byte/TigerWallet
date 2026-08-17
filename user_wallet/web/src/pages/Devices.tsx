// Devices Page — manage synced devices (real PG CRUD).
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

interface Device { id: string; name: string; device_type: string; status: string; last_sync?: string; }

export default function Devices() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [name, setName] = useState('');
  const [deviceType, setDeviceType] = useState('web');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    api.getDevices().then((data) => setDevices((data.devices as Device[]) || [])).catch(() => setDevices([])).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const register = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setBusy(true);
    try { await api.registerDevice({ name, deviceType }); setName(''); load(); } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed'); } finally { setBusy(false); }
  };

  const sync = async (id: string) => {
    setBusy(true);
    try { await api.syncDevice(id); load(); } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Sync failed'); } finally { setBusy(false); }
  };

  const remove = async (id: string) => {
    if (!window.confirm('Remove this device?')) return;
    setBusy(true);
    try { await api.deleteDevice(id); load(); } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Delete failed'); } finally { setBusy(false); }
  };

  return (
    <div className="devices-page">
      <h1>Devices</h1>
      {error && <div className="error">{error}</div>}
      <form onSubmit={register} className="device-form">
        <input placeholder="Device name" value={name} onChange={(e) => setName(e.target.value)} required />
        <select value={deviceType} onChange={(e) => setDeviceType(e.target.value)}>
          <option value="web">Web</option>
          <option value="desktop">Desktop</option>
          <option value="android">Android</option>
          <option value="ios">iOS</option>
          <option value="extension">Extension</option>
        </select>
        <button type="submit" disabled={busy}>Register</button>
      </form>
      {loading ? <p>Loading…</p> : devices.length === 0 ? <p>No registered devices.</p> : (
        <ul className="device-list">
          {devices.map((d) => (
            <li key={d.id}>
              <div><strong>{d.name}</strong> <span className={`status ${d.status}`}>{d.status}</span></div>
              <div className="small">{d.device_type}{d.last_sync ? ` · synced ${new Date(d.last_sync).toLocaleString()}` : ''}</div>
              <button onClick={() => sync(d.id)}>Sync</button>
              <button onClick={() => remove(d.id)}>Delete</button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
