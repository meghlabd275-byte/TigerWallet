'use client';

import React, { useState, useEffect } from 'react';

export default function BiometricAuth() {
  const [biometricType, setBiometricType] = useState<string | null>(null);
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    const types = ['Face ID', 'Fingerprint', 'Touch ID', 'Windows Hello'];
    setBiometricType(types[Math.floor(Math.random() * types.length)]);
  }, []);

  const handleEnable = async () => {
    setLoading(true);
    await new Promise(r => setTimeout(r, 1500));
    setEnabled(true);
    setMessage(`${biometricType} enabled!`);
    setLoading(false);
  };

  const handleDisable = async () => {
    setLoading(true);
    await new Promise(r => setTimeout(r, 500));
    setEnabled(false);
    setMessage('Disabled');
    setLoading(false);
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white">
      <header className="bg-white dark:bg-slate-800 border-b p-4"><div className="flex items-center gap-4"><a href="/wallet" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Biometric Security</h1></div></header>
      <div className="max-w-md mx-auto p-8">
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6">
          <div className="text-center mb-6"><div className="text-6xl mb-4">🔐</div><h2 className="text-xl font-semibold">Biometric Authentication</h2><p className="text-slate-500 mt-2">Use {biometricType || 'biometric'} to secure</p></div>
          {biometricType && <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4 mb-6"><div className="flex items-center justify-between"><div className="flex items-center gap-3"><span className="text-3xl">👆</span><div><div className="font-semibold">{biometricType}</div><div className="text-xs text-slate-500">Available</div></div></div><div className="w-3 h-3 bg-green-500 rounded-full"></div></div></div>}
          {message && <div className="bg-green-100 text-green-600 p-3 rounded-lg mb-4 text-center">{message}</div>}
          {enabled ? <button onClick={handleDisable} disabled={loading} className="w-full bg-red-500 text-white py-3 rounded-lg">{loading ? '...' : 'Disable'}</button> : <button onClick={handleEnable} disabled={loading} className="w-full bg-orange-500 text-white py-3 rounded-lg">{loading ? '...' : 'Enable'}</button>}
        </div>
      </div>
    </div>
  );
}
