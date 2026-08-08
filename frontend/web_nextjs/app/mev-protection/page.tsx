'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

export default function MEVProtectionPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [protection, setProtection] = useState({
    flashbots: true,
    privatePool: true,
    mevBlocker: false,
    stealthSlippage: true,
  });

  return (
    <div className={`min-h-screen p-8 ${isDark ? 'bg-gradient-to-br from-slate-900 to-slate-800' : 'bg-gradient-to-br from-slate-50 to-slate-100'}`}>
      <div className="max-w-4xl mx-auto">
        <h1 className={`text-4xl font-bold mb-2 ${isDark ? 'text-white' : 'text-slate-900'}`}>MEV Protection</h1>
        <p className={`mb-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Protect your transactions from MEV extraction</p>

        <div className={`rounded-2xl p-6 border mb-6 ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
          <div className="flex items-center gap-4 mb-4">
            <span className="text-4xl">🛡️</span>
            <div>
              <h2 className={`text-xl font-semibold ${isDark ? 'text-white' : 'text-slate-900'}`}>MEV Shield Active</h2>
              <p className="text-green-400">Your transactions are protected</p>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div className={`p-3 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
              <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>$12,450</p>
              <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Saved this month</p>
            </div>
            <div className={`p-3 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
              <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>156</p>
              <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Protected TXs</p>
            </div>
            <div className={`p-3 rounded-lg ${isDark ? 'bg-slate-700/50' : 'bg-slate-100'}`}>
              <p className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-slate-900'}`}>2.3%</p>
              <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Avg. Savings</p>
            </div>
          </div>
        </div>

        <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-slate-900'}`}>Protection Features</h2>
        <div className="space-y-4">
          {[
            { key: 'flashbots', name: 'Flashbots Protect', desc: 'Private transaction relay' },
            { key: 'privatePool', name: 'Private Pools', desc: 'Hide transactions from public mempool' },
            { key: 'mevBlocker', name: 'MEV Blocker', desc: 'Block sandwich attacks' },
            { key: 'stealthSlippage', name: 'Stealth Slippage', desc: 'Dynamic slippage protection' },
          ].map(item => (
            <div key={item.key} className={`rounded-xl p-4 border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-slate-200'}`}>
              <div className="flex items-center justify-between">
                <div>
                  <h3 className={`font-medium ${isDark ? 'text-white' : 'text-slate-900'}`}>{item.name}</h3>
                  <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{item.desc}</p>
                </div>
                <button
                  onClick={() => setProtection(prev => ({ ...prev, [item.key]: !prev[item.key as keyof typeof prev] }))}
                  className={`w-14 h-8 rounded-full transition-colors ${protection[item.key as keyof typeof protection] ? 'bg-green-500' : 'bg-slate-600'}`}
                >
                  <div className={`w-6 h-6 bg-white rounded-full shadow transition-transform ${protection[item.key as keyof typeof protection] ? 'translate-x-7' : 'translate-x-1'}`} />
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
