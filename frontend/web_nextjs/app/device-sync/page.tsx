'use client';

import React, { useState } from 'react';

interface Device {
  id: string;
  name: string;
  type: 'mobile' | 'desktop' | 'tablet';
  lastSync: number;
  status: 'online' | 'offline';
}

export default function DeviceSyncPage() {
  const [devices, setDevices] = useState<Device[]>([
    { id: '1', name: 'iPhone 15 Pro', type: 'mobile', lastSync: Date.now() - 300000, status: 'online' },
    { id: '2', name: 'MacBook Pro', type: 'desktop', lastSync: Date.now() - 3600000, status: 'offline' },
    { id: '3', name: 'iPad Air', type: 'tablet', lastSync: Date.now() - 86400000, status: 'offline' },
  ]);
  const [syncing, setSyncing] = useState<string | null>(null);

  const syncDevice = async (id: string) => {
    setSyncing(id);
    await new Promise(r => setTimeout(r, 2000));
    setDevices(prev => prev.map(d => 
      d.id === id ? { ...d, lastSync: Date.now(), status: 'online' } : d
    ));
    setSyncing(null);
  };

  const removeDevice = (id: string) => {
    setDevices(prev => prev.filter(d => d.id !== id));
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
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-4xl font-bold text-white mb-2">Device Sync</h1>
        <p className="text-slate-400 mb-8">Manage your connected devices</p>

        <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700 mb-6">
          <div className="flex items-center gap-4">
            <div className="flex-1">
              <p className="text-white font-medium">Sync Status</p>
              <p className="text-slate-400 text-sm">All devices synchronized</p>
            </div>
            <span className="text-green-400 text-2xl">✓</span>
          </div>
        </div>

        <h2 className="text-xl font-semibold text-white mb-4">Connected Devices</h2>
        <div className="space-y-4">
          {devices.map(device => (
            <div key={device.id} className="bg-slate-800 rounded-xl p-4 border border-slate-700">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-3xl">{getDeviceIcon(device.type)}</span>
                  <div>
                    <h3 className="text-white font-medium">{device.name}</h3>
                    <p className="text-slate-400 text-sm">
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

        <div className="mt-8 p-4 bg-slate-800 rounded-xl border border-slate-700">
          <h3 className="text-white font-medium mb-4">Sync Settings</h3>
          <div className="space-y-3">
            {[
              { name: 'Auto-sync', desc: 'Automatically sync across devices' },
              { name: 'Sync transactions', desc: 'Include transaction history' },
              { name: 'Sync contacts', desc: 'Sync address book' },
            ].map((item, i) => (
              <label key={i} className="flex items-center justify-between cursor-pointer">
                <span className="text-white">{item.name}</span>
                <input type="checkbox" defaultChecked className="w-5 h-5 rounded" />
              </label>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
