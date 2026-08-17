/**
 * Devices Page - Multi-device sync management.
 *
 * Fetches registered devices (GET /devices), supports registering a new
 * device (POST /devices), syncing (POST /devices/:id/sync), and deleting
 * (DELETE /devices/:id). All calls go through WalletService; no mock data.
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService } from '../services/WalletService';
import LoadingSpinner from '../components/LoadingSpinner';

interface Device {
  id: string;
  name: string;
  device_type?: string;
  last_synced_at?: string;
  status?: string;
}

const DEVICE_TYPES = ['desktop', 'mobile', 'tablet', 'hardware', 'extension'];

function DevicesPage() {
  const { theme } = useTheme();
  const [walletService] = useState(() => new WalletService());

  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Register form state
  const [name, setName] = useState('');
  const [deviceType, setDeviceType] = useState(DEVICE_TYPES[0]);
  const [submitting, setSubmitting] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);

  const loadDevices = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = (await walletService.getDevices()) as Device[] | { devices?: Device[] };
      const list = Array.isArray(data) ? data : (data?.devices ?? []);
      setDevices(
        (list ?? []).map((d) => ({
          id: String(d.id ?? ''),
          name: String(d.name ?? ''),
          device_type: d.device_type ? String(d.device_type) : undefined,
          last_synced_at: d.last_synced_at ? String(d.last_synced_at) : undefined,
          status: d.status ? String(d.status) : undefined,
        }))
      );
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to load devices');
      setDevices([]);
    } finally {
      setLoading(false);
    }
  }, [walletService]);

  useEffect(() => {
    loadDevices();
  }, [loadDevices]);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!name.trim()) {
      setError('Device name is required');
      return;
    }
    setSubmitting(true);
    try {
      await walletService.registerDevice({ name: name.trim(), deviceType });
      setName('');
      await loadDevices();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to register device');
    } finally {
      setSubmitting(false);
    }
  };

  const handleSync = async (id: string) => {
    setError(null);
    setBusyId(id);
    try {
      await walletService.syncDevice(id);
      await loadDevices();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to sync device');
    } finally {
      setBusyId(null);
    }
  };

  const handleDelete = async (id: string) => {
    setError(null);
    setBusyId(id);
    try {
      await walletService.deleteDevice(id);
      await loadDevices();
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to delete device');
    } finally {
      setBusyId(null);
    }
  };

  const fmtTime = (ts?: string) => {
    if (!ts) return 'Never';
    const n = Number(ts);
    const d = n > 1e9 ? new Date(n * 1000) : new Date(ts);
    return isNaN(d.getTime()) ? ts : d.toLocaleString();
  };

  const cardClass = `card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`;
  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Devices</h1>

      {error && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <p className="text-sm text-red-500">{error}</p>
        </div>
      )}

      {/* Register device form */}
      <form onSubmit={handleRegister} className={cardClass}>
        <h3 className="font-semibold mb-4">Register Device</h3>

        <div className="mb-4">
          <label className="label">Device Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Laptop"
            className={inputClass}
            required
          />
        </div>

        <div className="mb-6">
          <label className="label">Device Type</label>
          <select
            value={deviceType}
            onChange={(e) => setDeviceType(e.target.value)}
            className={inputClass}
          >
            {DEVICE_TYPES.map((t) => (
              <option key={t} value={t}>
                {t.replace(/\b\w/g, (c) => c.toUpperCase())}
              </option>
            ))}
          </select>
        </div>

        <button type="submit" disabled={submitting} className="btn btn-primary w-full">
          {submitting ? 'Registering...' : 'Register Device'}
        </button>
      </form>

      {/* Devices list */}
      <h3 className="font-semibold mb-3">Registered Devices</h3>
      {loading ? (
        <LoadingSpinner label="Loading devices..." />
      ) : devices.length === 0 ? (
        <div className={`card text-center py-12 opacity-60 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
          No devices registered.
        </div>
      ) : (
        <div className="space-y-3">
          {devices.map((d) => (
            <div key={d.id} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start gap-3">
                <div className="min-w-0">
                  <div className="font-semibold flex items-center gap-2">
                    {d.name}
                    {d.device_type && (
                      <span className={`text-xs px-2 py-0.5 rounded ${theme === 'dark' ? 'bg-slate-700' : 'bg-gray-200'}`}>
                        {d.device_type}
                      </span>
                    )}
                  </div>
                  <p className="text-xs opacity-60 mt-1">Last synced: {fmtTime(d.last_synced_at)}</p>
                  {d.status && <p className="text-xs opacity-40 mt-1">Status: {d.status}</p>}
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleSync(d.id)}
                    disabled={busyId === d.id}
                    className="btn btn-secondary text-sm"
                  >
                    {busyId === d.id ? 'Syncing...' : 'Sync'}
                  </button>
                  <button
                    onClick={() => handleDelete(d.id)}
                    disabled={busyId === d.id}
                    className="btn btn-secondary text-sm"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default DevicesPage;
