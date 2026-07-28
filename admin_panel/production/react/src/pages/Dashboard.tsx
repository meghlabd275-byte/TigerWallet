/**
 * Admin Dashboard - Super Admin Panel
 */

import React from 'react';
import { useNavigate } from 'react-router-dom';

function Dashboard() {
  const navigate = useNavigate();

  const stats = [
    { label: 'Total Users', value: '12,456', change: '+12%' },
    { label: 'Total Volume', value: '$45.2M', change: '+8%' },
    { label: 'Active Wallets', value: '8,234', change: '+15%' },
    { label: 'Transactions', value: '156,789', change: '+22%' },
  ];

  const modules = [
    { name: 'User Management', path: '/users', icon: '👥', desc: 'Manage all platform users' },
    { name: 'White Label Clients', path: '/whitelabel', icon: '🏢', desc: 'White label client management' },
    { name: 'Token Management', path: '/tokens', icon: '🪙', desc: 'Manage tokens & cryptocurrencies' },
    { name: 'Pair Management', path: '/pairs', icon: '🔄', desc: 'Trading pair configuration' },
    { name: 'Liquidity Management', path: '/liquidity', icon: '💧', desc: 'Liquidity pool management' },
    { name: 'Fee Management', path: '/fees', icon: '💰', desc: 'Platform fee configuration' },
    { name: 'Blockchain Management', path: '/blockchains', icon: '⛓️', desc: 'Add/remove blockchain networks' },
    { name: 'KYC Management', path: '/kyc', icon: '🛡️', desc: 'User verification' },
    { name: 'Master Wallet', path: '/master-wallet', icon: '🔐', desc: 'Master wallet operations' },
    { name: 'Analytics', path: '/analytics', icon: '📊', desc: 'Platform analytics' },
    { name: 'API Management', path: '/api', icon: '🔌', desc: 'API keys & access' },
    { name: 'Settings', path: '/settings', icon: '⚙️', desc: 'Platform settings' },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Super Admin Dashboard</h1>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        {stats.map((stat, i) => (
          <div key={i} className="bg-slate-800 p-4 rounded-lg">
            <p className="text-sm opacity-60">{stat.label}</p>
            <p className="text-2xl font-bold">{stat.value}</p>
            <p className="text-sm text-green-500">{stat.change}</p>
          </div>
        ))}
      </div>

      {/* Modules Grid */}
      <h2 className="text-xl font-semibold mb-4">Management Modules</h2>
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {modules.map((mod, i) => (
          <div
            key={i}
            onClick={() => navigate(mod.path)}
            className="bg-slate-800 p-4 rounded-lg cursor-pointer hover:bg-slate-700 transition"
          >
            <div className="text-3xl mb-2">{mod.icon}</div>
            <h3 className="font-semibold">{mod.name}</h3>
            <p className="text-sm opacity-60">{mod.desc}</p>
          </div>
        ))}
      </div>

      {/* Recent Activity */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold mb-4">Recent Activity</h2>
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Time</th>
                <th className="p-3 text-left">Action</th>
                <th className="p-3 text-left">Admin</th>
                <th className="p-3 text-left">Details</th>
              </tr>
            </thead>
            <tbody>
              {[
                { time: '2 min ago', action: 'Create Token', admin: 'admin@tiger', details: 'Created USDT token' },
                { time: '15 min ago', action: 'Update Fee', admin: 'admin@tiger', details: 'Changed swap fee to 0.3%' },
                { time: '1 hour ago', action: 'Add Chain', admin: 'admin@tiger', details: 'Added Arbitrum One' },
                { time: '3 hours ago', action: 'Suspend User', admin: 'superadmin', details: 'Suspended user #45892' },
              ].map((item, i) => (
                <tr key={i} className="border-t border-slate-700">
                  <td className="p-3">{item.time}</td>
                  <td className="p-3 text-amber-500">{item.action}</td>
                  <td className="p-3">{item.admin}</td>
                  <td className="p-3 opacity-60">{item.details}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
