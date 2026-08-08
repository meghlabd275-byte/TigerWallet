'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface PrivacyFeature {
  id: string;
  name: string;
  description: string;
  icon: string;
  enabled: boolean;
}

const PRIVACY_FEATURES: PrivacyFeature[] = [
  { id: 'stealth', name: 'Stealth Addresses', description: 'Generate one-time addresses for each transaction', icon: '🎭', enabled: false },
  { id: 'mixer', name: 'Privacy Mixer', description: 'Break on-chain transaction links', icon: '🔀', enabled: false },
  { id: 'shield', name: 'Shielded Transactions', description: 'Private transactions with zero-knowledge proofs', icon: '🛡️', enabled: false },
  { id: 'vpn', name: 'VPN Integration', description: 'Route through VPN for IP privacy', icon: '🔒', enabled: false },
  { id: 'relay', name: 'TX Relay', description: 'Hide your address from recipients', icon: '📡', enabled: false },
  { id: 'hide', name: 'Balance Hiding', description: 'Hide token balances from block explorers', icon: '👁️‍🗨️', enabled: false },
];

export default function PrivacyPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [features, setFeatures] = useState(PRIVACY_FEATURES);
  const [isLoading, setIsLoading] = useState(false);

  const toggleFeature = (id: string) => {
    setFeatures(prev => prev.map(f => 
      f.id === id ? { ...f, enabled: !f.enabled } : f
    ));
  };

  const enabledCount = features.filter(f => f.enabled).length;

  return (
    <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
      <div className="max-w-4xl mx-auto">
        <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>Privacy Features</h1>
        <p className={`mb-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Protect your financial privacy</p>

        <div className={`rounded-2xl p-6 border mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className={`text-xl font-semibold ${isDark ? 'text-white' : 'text-slate-900'}`}>Privacy Shield</h2>
              <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{enabledCount} of {features.length} features enabled</p>
            </div>
            <div className={`w-32 h-2 rounded-full overflow-hidden ${isDark ? 'bg-slate-700' : 'bg-slate-200'}`}>
              <div
                className="h-full bg-gradient-to-r from-green-500 to-blue-500 transition-all"
                style={{ width: `${(enabledCount / features.length) * 100}%` }}
              />
            </div>
          </div>
        </div>

        <div className="grid gap-4">
          {features.map(feature => (
            <div key={feature.id} className={`rounded-xl p-4 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <span className="text-3xl">{feature.icon}</span>
                  <div>
                    <h3 className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{feature.name}</h3>
                    <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{feature.description}</p>
                  </div>
                </div>
                <button
                  onClick={() => toggleFeature(feature.id)}
                  className={`w-14 h-8 rounded-full transition-colors ${
                    feature.enabled ? 'bg-green-500' : 'bg-slate-600'
                  }`}
                >
                  <div className={`w-6 h-6 bg-white rounded-full shadow transition-transform ${
                    feature.enabled ? 'translate-x-7' : 'translate-x-1'
                  }`} />
                </button>
              </div>
            </div>
          ))}
        </div>

        <div className="mt-8 p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-xl">
          <p className="text-yellow-400 text-sm">
            ⚠️ Some privacy features may have regulatory implications in certain jurisdictions. 
            Use responsibly and ensure compliance with local laws.
          </p>
        </div>
      </div>
    </div>
  );
}
