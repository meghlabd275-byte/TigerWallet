'use client';

import React, { useState } from 'react';

export default function MEVProtectionPage() {
  const [protection, setProtection] = useState({
    flashbots: true,
    privatePool: true,
    mevBlocker: false,
    stealthSlippage: true,
  });

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-4xl font-bold text-white mb-2">MEV Protection</h1>
        <p className="text-slate-400 mb-8">Protect your transactions from MEV extraction</p>

        <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700 mb-6">
          <div className="flex items-center gap-4 mb-4">
            <span className="text-4xl">🛡️</span>
            <div>
              <h2 className="text-xl font-semibold text-white">MEV Shield Active</h2>
              <p className="text-green-400">Your transactions are protected</p>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div className="p-3 bg-slate-700/50 rounded-lg">
              <p className="text-2xl font-bold text-white">$12,450</p>
              <p className="text-slate-400 text-sm">Saved this month</p>
            </div>
            <div className="p-3 bg-slate-700/50 rounded-lg">
              <p className="text-2xl font-bold text-white">156</p>
              <p className="text-slate-400 text-sm">Protected TXs</p>
            </div>
            <div className="p-3 bg-slate-700/50 rounded-lg">
              <p className="text-2xl font-bold text-white">2.3%</p>
              <p className="text-slate-400 text-sm">Avg. Savings</p>
            </div>
          </div>
        </div>

        <h2 className="text-xl font-semibold text-white mb-4">Protection Features</h2>
        <div className="space-y-4">
          {[
            { key: 'flashbots', name: 'Flashbots Protect', desc: 'Private transaction relay' },
            { key: 'privatePool', name: 'Private Pools', desc: 'Hide transactions from public mempool' },
            { key: 'mevBlocker', name: 'MEV Blocker', desc: 'Block sandwich attacks' },
            { key: 'stealthSlippage', name: 'Stealth Slippage', desc: 'Dynamic slippage protection' },
          ].map(item => (
            <div key={item.key} className="bg-slate-800 rounded-xl p-4 border border-slate-700">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-white font-medium">{item.name}</h3>
                  <p className="text-slate-400 text-sm">{item.desc}</p>
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
