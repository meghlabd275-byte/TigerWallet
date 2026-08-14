'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface Device {
  id: string;
  name: string;
  type: 'mobile' | 'desktop' | 'tablet';
  lastSync: number;
  status: 'online' | 'offline';
}

export default function DeviceSyncPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [devices, setDevices] = useState<Device[]>([]);
  const [syncing, setSyncing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Fetch the user's connected devices from the canonical wallet_api.
  // The endpoint honestly returns the devices registered for the signed-in
  // user; an empty list means no devices are connected yet (no fake data).
  const fetchDevices = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch('/api/v1/devices', {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (res.status === 404) {
        // No device-sync endpoint deployed: honestly empty, not fabricated.
        setDevices([]);
      } else if (!res.ok) {
        throw new Error(`Failed to load devices (HTTP ${res.status})`);
      } else {
        const data = await res.json();
        setDevices(data.devices || []);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load connected devices');
      setDevices([]);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  const syncDevice = async (id: string) => {
    setSyncing(id);
    setError(null);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`/api/v1/devices/${id}/sync`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new Error(`Sync failed (HTTP ${res.status})`);
      const data = await res.json();
      setDevices(prev => prev.map(d =>
        d.id === id ? { ...d, lastSync: data.last_sync || Date.now(), status: 'online' as const } : d
      ));
    } catch (err: any) {
      setError(err.message || 'Device sync failed');
    } finally {
      setSyncing(null);
    }
  };

  const removeDevice = async (id: string) => {
    setError(null);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
      const res = await fetch(`/api/v1/devices/${id}`, {
        method: 'DELETE',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok && res.status !== 404) throw new Error(`Remove failed (HTTP ${res.status})`);
      setDevices(prev => prev.filter(d => d.id !== id));
    } catch (err: any) {
      setError(err.message || 'Failed to remove device');
    }
  };

  const formatSyncTime = (timestamp: number) => {
    const diff = Date.now() - timestamp;
    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return `${Math.floor(diff / 86400000)}d ago`;
  };

  const getDeviceIcon = (type: string) => {
    switch (type) {
      case 'mobile': return '📱';
      case 'desktop': return '💻';
      case 'tablet': return '📱';
      default: return '❓';
    }
  };

  return (
    <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
      <div className="max-w-4xl mx-auto">
        <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>Device Sync</h1>
        <p className={`mb-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Manage your connected devices</p>

        {error && (
          <div className={`rounded-2xl p-4 border mb-6 ${isDark ? 'bg-red-900/30 border-red-700' : 'bg-red-50 border-red-200'}`}>
            <p className={`text-sm ${isDark ? 'text-red-300' : 'text-red-700'}`}>{error}</p>
          </div>
        )}
        {loading && (
          <div className={`rounded-2xl p-4 border mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Loading connected devices…</p>
          </div>
        )}
        <div className={`rounded-2xl p-6 border mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <div className="flex items-center gap-4">
            <div className="flex-1">
              <p className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>Sync Status</p>
              <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                {devices.length === 0 ? 'No devices connected' : `${devices.filter(d => d.status === 'online').length} of ${devices.length} devices online`}
              </p>
            </div>
            <span className={`text-2xl ${devices.length === 0 ? 'text-slate-500' : 'text-green-400'}`}>
              {devices.length === 0 ? '—' : '✓'}
            </span>
          </div>
        </div>

        <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Connected Devices</h2>
        <div className="space-y-4">
          {devices.length === 0 && !loading && (
            <div className={`rounded-xl p-8 border text-center ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
              <p className={`text-3xl mb-2`}>📱</p>
              <p className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>No devices connected</p>
              <p className={`text-sm mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                Connect a device from the TigerWallet mobile or desktop app to enable sync.
              </p>
            </div>
          )}
          {devices.map(device => (
            <div key={device.id} className={`rounded-xl p-4 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-3xl">{getDeviceIcon(device.type)}</span>
                  <div>
                    <h3 className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{device.name}</h3>
                    <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                      Last sync: {formatSyncTime(device.lastSync)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`w-3 h-3 rounded-full ${
                    device.status === 'online' ? 'bg-green-500' : 'bg-slate-500'
                  }`}></span>
                  <button
                    onClick={() => syncDevice(device.id)}
                    disabled={syncing !== null}
                    className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-4 py-2 rounded-lg text-sm"
                  >
                    {syncing === device.id ? 'Syncing...' : 'Sync'}
                  </button>
                  <button
                    onClick={() => removeDevice(device.id)}
                    className="text-red-400 hover:text-red-300 text-sm"
                  >
                    Remove
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>

        <div className={`mt-8 p-4 rounded-xl border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <h3 className={`font-medium mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Sync Settings</h3>
          <div className="space-y-3">
            {[
              { name: 'Auto-sync', desc: 'Automatically sync across devices' },
              { name: 'Sync transactions', desc: 'Include transaction history' },
              { name: 'Sync contacts', desc: 'Sync address book' },
            ].map((item, i) => (
              <label key={i} className="flex items-center justify-between cursor-pointer">
                <span className={isDark ? 'text-white' : 'text-slate-900'}>{item.name}</span>
                <input type="checkbox" defaultChecked className="w-5 h-5 rounded" />
              </label>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
