/**
 * API Management
 */

import React, { useState } from 'react';

function APIManagement() {
  const [activeTab, setActiveTab] = useState('keys');

  const apiKeys = [
    { id: 1, name: 'Production Key', key: 'tk_live_xxxx...xxxx', created: '2024-01-01', lastUsed: '2 hours ago', status: 'Active', rateLimit: '1000/min' },
    { id: 2, name: 'Development Key', key: 'tk_test_xxxx...xxxx', created: '2024-01-10', lastUsed: '5 days ago', status: 'Active', rateLimit: '100/min' },
    { id: 3, name: 'Mobile App', key: 'tk_mobile_xxxx...xxxx', created: '2024-01-15', lastUsed: '1 hour ago', status: 'Active', rateLimit: '500/min' },
  ];

  const endpoints = [
    { method: 'GET', path: '/api/v1/wallets', description: 'Get user wallets', rateLimit: '100/min' },
    { method: 'POST', path: '/api/v1/wallets/create', description: 'Create new wallet', rateLimit: '10/min' },
    { method: 'POST', path: '/api/v1/transactions/send', description: 'Send transaction', rateLimit: '50/min' },
    { method: 'GET', path: '/api/v1/balances/:address', description: 'Get balance', rateLimit: '200/min' },
    { method: 'GET', path: '/api/v1/prices', description: 'Get token prices', rateLimit: '100/min' },
  ];

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">API Management</h1>
        <button className="btn btn-primary">+ Generate New Key</button>
      </div>

      <div className="flex gap-2 mb-6">
        {['Keys', 'Endpoints', 'Webhooks', 'Settings'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'keys' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Name</th>
                <th className="p-3 text-left">API Key</th>
                <th className="p-3 text-left">Created</th>
                <th className="p-3 text-left">Last Used</th>
                <th className="p-3 text-left">Rate Limit</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {apiKeys.map(key => (
                <tr key={key.id} className="border-t border-slate-700">
                  <td className="p-3 font-semibold">{key.name}</td>
                  <td className="p-3 font-mono text-sm">{key.key}</td>
                  <td className="p-3">{key.created}</td>
                  <td className="p-3">{key.lastUsed}</td>
                  <td className="p-3">{key.rateLimit}</td>
                  <td className="p-3">
                    <span className="px-2 py-1 rounded text-xs bg-green-500/20 text-green-500">{key.status}</span>
                  </td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Regenerate</button>
                      <button className="btn btn-danger text-xs">Revoke</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'endpoints' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Method</th>
                <th className="p-3 text-left">Endpoint</th>
                <th className="p-3 text-left">Description</th>
                <th className="p-3 text-left">Rate Limit</th>
              </tr>
            </thead>
            <tbody>
              {endpoints.map((ep, i) => (
                <tr key={i} className="border-t border-slate-700">
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${
                      ep.method === 'GET' ? 'bg-blue-500/20 text-blue-500' :
                      ep.method === 'POST' ? 'bg-green-500/20 text-green-500' :
                      ep.method === 'PUT' ? 'bg-yellow-500/20 text-yellow-500' :
                      'bg-red-500/20 text-red-500'
                    }`}>{ep.method}</span>
                  </td>
                  <td className="p-3 font-mono text-sm">{ep.path}</td>
                  <td className="p-3">{ep.description}</td>
                  <td className="p-3">{ep.rateLimit}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'webhooks' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Webhook Endpoints</h3>
          <p className="text-sm opacity-60 mb-4">Configure webhook URLs for real-time events:</p>
          <div className="space-y-4 max-w-lg">
            <div>
              <label className="label">Transaction Webhook URL</label>
              <input type="text" placeholder="https://..." className="input" />
            </div>
            <div>
              <label className="label">Wallet Webhook URL</label>
              <input type="text" placeholder="https://..." className="input" />
            </div>
            <button className="btn btn-primary">Save Webhooks</button>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">API Settings</h3>
          <div className="space-y-4 max-w-lg">
            <div>
              <label className="label">Default Rate Limit</label>
              <select className="input">
                <option>100 requests/minute</option>
                <option>500 requests/minute</option>
                <option>1000 requests/minute</option>
                <option>10000 requests/minute</option>
              </select>
            </div>
            <div className="flex items-center gap-2">
              <input type="checkbox" id="ipWhitelist" />
              <label htmlFor="ipWhitelist">Enable IP Whitelist</label>
            </div>
            <button className="btn btn-primary">Save Settings</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default APIManagement;
