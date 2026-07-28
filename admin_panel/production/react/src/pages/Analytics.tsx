/**
 * Analytics Dashboard
 */

import React, { useState } from 'react';

function Analytics() {
  const [timeRange, setTimeRange] = useState('7d');

  const metrics = [
    { label: 'Total Users', value: '12,456', change: '+12%' },
    { label: 'Active Users', value: '8,234', change: '+8%' },
    { label: 'Total Volume', value: '$45.2M', change: '+15%' },
    { label: 'Revenue', value: '$1.2M', change: '+22%' },
    { label: 'Transactions', value: '156,789', change: '+18%' },
    { label: 'Gas Saved', value: '$234K', change: '+30%' },
  ];

  const topTokens = [
    { name: 'Ethereum', volume: '$12.5M', trades: 45000 },
    { name: 'Bitcoin', volume: '$8.2M', trades: 23000 },
    { name: 'USDT', volume: '$15.3M', trades: 89000 },
    { name: 'BNB', volume: '$5.6M', trades: 34000 },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Analytics</h1>
        <div className="flex gap-2">
          {['24h', '7d', '30d', '90d', '1y'].map(range => (
            <button key={range} onClick={() => setTimeRange(range)} className={`px-3 py-1 rounded ${timeRange === range ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
              {range}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-6">
        {metrics.map((m, i) => (
          <div key={i} className="bg-slate-800 p-4 rounded-lg">
            <p className="text-sm opacity-60">{m.label}</p>
            <p className="text-xl font-bold">{m.value}</p>
            <p className="text-sm text-green-500">{m.change}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Volume Over Time</h3>
          <div className="h-48 flex items-end justify-between gap-2">
            {[65, 45, 78, 56, 89, 67, 45, 78, 56, 89, 67, 45].map((h, i) => (
              <div key={i} className="flex-1 bg-amber-500/50 rounded-t" style={{height: `${h}%`}}></div>
            ))}
          </div>
        </div>

        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Top Tokens by Volume</h3>
          <div className="space-y-3">
            {topTokens.map((t, i) => (
              <div key={i} className="flex justify-between items-center">
                <span>{t.name}</span>
                <div className="text-right">
                  <p className="font-semibold">{t.volume}</p>
                  <p className="text-sm opacity-60">{t.trades.toLocaleString()} trades</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mt-6">
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">User Growth</h3>
          <p className="text-3xl font-bold text-green-500">+12%</p>
          <p className="text-sm opacity-60">vs last period</p>
        </div>
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Avg Transaction Size</h3>
          <p className="text-3xl font-bold">$287</p>
          <p className="text-sm opacity-60">+5% vs last period</p>
        </div>
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Platform Fees</h3>
          <p className="text-3xl font-bold text-amber-500">$1.2M</p>
          <p className="text-sm opacity-60">+22% vs last period</p>
        </div>
      </div>
    </div>
  );
}

export default Analytics;
