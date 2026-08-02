/**
 * Token Management - Complete Token Management
 * Connected to backend APIs
 */

import React, { useState, useEffect, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface Token {
  id: string;
  symbol: string;
  name: string;
  chain: string;
  decimals: number;
  address: string;
  status: 'active' | 'paused' | 'disabled';
  balance: number;
  logoUrl?: string;
}

function TokenManagement() {
  const [activeTab, setActiveTab] = useState('tokens');
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch tokens from backend
  const fetchTokens = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/tokens`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch tokens');
      }
      
      const data = await response.json();
      setTokens(data.tokens || []);
    } catch (err) {
      console.error('Error fetching tokens:', err);
      setError(err instanceof Error ? err.message : 'Failed to load tokens');
      setTokens([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTokens();
  }, [fetchTokens]);

  // Update token status
  const handleStatusChange = async (tokenId: string, newStatus: 'active' | 'paused' | 'disabled') => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/tokens/${tokenId}/status`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: newStatus }),
      });
      
      if (!response.ok) {
        throw new Error('Failed to update token status');
      }
      
      await fetchTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Status update failed');
    }
  };

  // Delete token
  const handleDeleteToken = async (tokenId: string) => {
    if (!confirm('Are you sure you want to delete this token?')) return;
    
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/tokens/${tokenId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to delete token');
      }
      
      await fetchTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed');
    }
  };

  const addToken = () => {
    console.log('Add token');
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Token Management</h1>
        <button onClick={addToken} className="btn btn-primary">+ Add Token</button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {['Tokens', 'Create Token', 'Import', 'Settings'].map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab.toLowerCase().replace(' ', ''))}
            className={`px-4 py-2 rounded-lg ${
              activeTab === tab.toLowerCase().replace(' ', '') ? 'bg-amber-500 text-black' : 'bg-slate-800'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tokens Table */}
      {activeTab === 'tokens' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Symbol</th>
                <th className="p-3 text-left">Name</th>
                <th className="p-3 text-left">Chain</th>
                <th className="p-3 text-left">Decimals</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Balance</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {tokens.map((token, i) => (
                <tr key={i} className="border-t border-slate-700">
                  <td className="p-3 font-bold text-amber-500">{token.symbol}</td>
                  <td className="p-3">{token.name}</td>
                  <td className="p-3">{token.chain}</td>
                  <td className="p-3">{token.decimals}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${
                      token.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-yellow-500/20 text-yellow-500'
                    }`}>
                      {token.status}
                    </span>
                  </td>
                  <td className="p-3">{token.balance}</td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-danger text-xs">{token.status === 'Active' ? 'Pause' : 'Activate'}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Token */}
      {activeTab === 'createtoken' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Create New Token</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Token Name</label>
              <input type="text" className="input" placeholder="My Token" />
            </div>
            <div>
              <label className="label">Symbol</label>
              <input type="text" className="input" placeholder="MTK" />
            </div>
            <div>
              <label className="label">Decimals</label>
              <input type="number" className="input" defaultValue={18} />
            </div>
            <div>
              <label className="label">Chain</label>
              <select className="input">
                <option>Ethereum</option>
                <option>BNB Chain</option>
                <option>Polygon</option>
                <option>Solana</option>
              </select>
            </div>
            <div>
              <label className="label">Initial Supply</label>
              <input type="number" className="input" placeholder="1000000" />
            </div>
            <div>
              <label className="label">Max Supply</label>
              <input type="number" className="input" placeholder="10000000" />
            </div>
          </div>
          <button className="btn btn-primary mt-4">Deploy Token</button>
        </div>
      )}
    </div>
  );
}

export default TokenManagement;
