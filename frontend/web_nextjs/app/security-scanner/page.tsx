'use client';

import React, { useState } from 'react';
import api, { ScanResult } from '../../src/lib/api/client';

export default function SecurityScannerPage() {
  const [address, setAddress] = useState('');
  const [scanning, setScanning] = useState(false);
  const [results, setResults] = useState<ScanResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const scan = async () => {
    if (!address) return;
    setScanning(true);
    setError(null);
    setResults(null);
    try {
      const res = await api.scanAddress(address);
      if (res.success && res.data) {
        setResults(res.data);
      } else {
        setError(res.error || 'Scan failed');
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Scan failed');
    } finally {
      setScanning(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-4xl font-bold text-white mb-2">Security Scanner</h1>
        <p className="text-slate-400 mb-8">Scan addresses and contracts for risks</p>

        <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700 mb-6">
          <div className="flex gap-4">
            <input
              type="text"
              value={address}
              onChange={(e) => setAddress(e.target.value)}
              placeholder="Enter address to scan..."
              className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
            />
            <button
              onClick={scan}
              disabled={!address || scanning}
              className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-8 py-3 rounded-lg font-medium"
            >
              {scanning ? 'Scanning...' : 'Scan'}
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-6 p-4 rounded-lg bg-red-900/50 border border-red-700 text-red-200">
            {error}
          </div>
        )}

        {results && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <div className="flex items-center gap-4 mb-6">
              <span className={`text-4xl ${
                results.risk === 'safe' ? 'text-green-500' :
                results.risk === 'warning' ? 'text-yellow-500' : 'text-red-500'
              }`}>
                {results.risk === 'safe' ? '✓' : results.risk === 'warning' ? '⚠' : '✕'}
              </span>
              <div>
                <h2 className="text-xl font-semibold text-white">
                  {results.risk === 'safe' ? 'Safe' : results.risk === 'warning' ? 'Warning' : 'Danger'}
                </h2>
                <p className="text-slate-400 text-sm">Scanned at {new Date(results.scannedAt).toLocaleString()}</p>
              </div>
            </div>

            {results.issues.length > 0 ? (
              <div className="space-y-3">
                <h3 className="text-white font-medium">Issues Found</h3>
                {results.issues.map((issue, i) => (
                  <div key={i} className="flex items-center gap-3 p-3 bg-slate-700/50 rounded-lg">
                    <span className="text-yellow-500">⚠</span>
                    <span className="text-slate-300">{issue}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-green-400">No issues detected.</p>
            )}
          </div>
        )}

        <div className="mt-8 grid grid-cols-3 gap-4">
          {[
            { name: 'Honeypot', icon: '🐝', desc: 'Detect fake tokens' },
            { name: 'Approvals', icon: '✓', desc: 'Review allowances' },
            { name: 'Simulation', icon: '🔮', desc: 'TX preview' },
          ].map(item => (
            <div key={item.name} className="bg-slate-800 rounded-xl p-4 border border-slate-700 text-center">
              <span className="text-2xl mb-2 block">{item.icon}</span>
              <h3 className="text-white font-medium">{item.name}</h3>
              <p className="text-slate-400 text-sm">{item.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
