/**
 * White Label Management - Complete Client Management
 */

import React, { useState } from 'react';

function WhiteLabelManagement() {
  const [activeTab, setActiveTab] = useState('clients');
  const [showCreate, setShowCreate] = useState(false);

  const clients = [
    { id: 1, name: 'CryptoCorp', domain: 'wallet.cryptocorp.io', status: 'Active', users: 1234, plan: 'Enterprise' },
    { id: 2, name: 'BankX', domain: 'pay.bankx.com', status: 'Active', users: 5678, plan: 'Business' },
    { id: 3, name: 'GameFi Hub', domain://gamefihub.io', status: 'Paused', users: 890, plan: 'Starter' },
    { id: 4, name: 'TradePro', domain: 'trade.pro', status: 'Active', users: 2345, plan: 'Enterprise' },
  ];

  const features = [
    { name: 'Custom Branding', enabled: true },
    { name: 'Custom Domain', enabled: true },
    { name: 'White Label Wallet', enabled: true },
    { name: 'Custom Trading Pairs', enabled: true },
    { name: 'API Access', enabled: true },
    { name: 'Dedicated Support', enabled: false },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">White Label Management</h1>
        <button onClick={() => setShowCreate(true)} className="btn btn-primary">
          + Create New Client
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {['Clients', 'Plans', 'Features', 'Analytics'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab.toLowerCase())}
            className={`px-4 py-2 rounded-lg ${
              activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Clients Tab */}
      {activeTab === 'clients' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">ID</th>
                <th className="p-3 text-left">Client Name</th>
                <th className="p-3 text-left">Domain</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Users</th>
                <th className="p-3 text-left">Plan</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {clients.map((client) => (
                <tr key={client.id} className="border-t border-slate-700">
                  <td className="p-3">#{client.id}</td>
                  <td className="p-3 font-semibold">{client.name}</td>
                  <td className="p-3 font-mono text-sm">{client.domain}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${
                      client.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-yellow-500/20 text-yellow-500'
                    }`}>
                      {client.status}
                    </span>
                  </td>
                  <td className="p-3">{client.users.toLocaleString()}</td>
                  <td className="p-3 text-amber-500">{client.plan}</td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-secondary text-xs">Manage</button>
                      <button className="btn btn-danger text-xs">Suspend</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Plans Tab */}
      {activeTab === 'plans' && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {['Starter', 'Business', 'Enterprise'].map((plan, i) => (
            <div key={plan} className="bg-slate-800 p-6 rounded-lg">
              <h3 className="text-xl font-bold mb-2">{plan}</h3>
              <p className="text-3xl font-bold text-amber-500 mb-4">
                ${(i + 1) * 999}<span className="text-sm opacity-60">/mo</span>
              </p>
              <ul className="space-y-2 mb-6">
                <li>✓ Up to {((i + 1) * 5000).toLocaleString()} users</li>
                <li>✓ Custom branding</li>
                <li>✓ Custom domain</li>
                <li>✓ {i + 1} Admin accounts</li>
                {i > 0 && <li>✓ API access</li>}
                {i > 1 && <li>✓ Dedicated support</li>}
              </ul>
              <button className="btn btn-primary w-full">Edit Plan</button>
            </div>
          ))}
        </div>
      )}

      {/* Features Tab */}
      {activeTab === 'features' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Platform Features</h3>
          <div className="grid grid-cols-2 gap-4">
            {features.map((feature, i) => (
              <div key={i} className="flex justify-between items-center p-3 bg-slate-700 rounded-lg">
                <span>{feature.name}</span>
                <button className={`w-12 h-6 rounded-full ${
                  feature.enabled ? 'bg-amber-500' : 'bg-slate-600'
                }`}>
                  <div className={`w-5 h-5 bg-white rounded-full transition ${
                    feature.enabled ? 'translate-x-6' : 'translate-x-0.5'
                  }`} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-800 p-6 rounded-lg w-full max-w-lg">
            <h2 className="text-xl font-bold mb-4">Create New White Label Client</h2>
            <div className="space-y-4">
              <div>
                <label className="label">Client Name</label>
                <input type="text" className="input" placeholder="Company Name" />
              </div>
              <div>
                <label className="label">Domain</label>
                <input type="text" className="input" placeholder="wallet.yourdomain.com" />
              </div>
              <div>
                <label className="label">Plan</label>
                <select className="input">
                  <option>Starter</option>
                  <option>Business</option>
                  <option>Enterprise</option>
                </select>
              </div>
              <div>
                <label className="label">Admin Email</label>
                <input type="email" className="input" placeholder="admin@company.com" />
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button onClick={() => setShowCreate(false)} className="btn btn-secondary flex-1">Cancel</button>
              <button className="btn btn-primary flex-1">Create Client</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default WhiteLabelManagement;
