/**
 * Blockchain Management - Add/manage blockchain networks
 * Connected to backend APIs
 */

import React, { useState, useEffect, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface Blockchain {
  id: string;
  name: string;
  symbol: string;
  chainId: number | null;
  rpc: string;
  explorer: string;
  type: 'EVM' | 'Non-EVM';
  status: 'active' | 'paused' | 'disabled';
}

function BlockchainManagement() {
  const [activeTab, setActiveTab] = useState('networks');
  const [blockchains, setBlockchains] = useState<Blockchain[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch blockchains from backend
  const fetchBlockchains = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/blockchains`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch blockchains');
      }
      
      const data = await response.json();
      setBlockchains(data.blockchains || []);
    } catch (err) {
      console.error('Error fetching blockchains:', err);
      setError(err instanceof Error ? err.message : 'Failed to load blockchains');
      setBlockchains([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchBlockchains();
  }, [fetchBlockchains]);

  // Update blockchain status
  const handleStatusChange = async (chainId: string, newStatus: 'active' | 'paused' | 'disabled') => {
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/blockchains/${chainId}/status`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: newStatus }),
      });
      
      if (!response.ok) {
        throw new Error('Failed to update blockchain status');
      }
      
      await fetchBlockchains();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Status update failed');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500"></div>
      </div>
    );
  }

  const evmChains = blockchains.filter(b => b.type === 'EVM').length;
  const nonEvmChains = blockchains.filter(b => b.type === 'Non-EVM').length;
  const activeChains = blockchains.filter(b => b.status === 'active').length;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Blockchain Management</h1>
      </div>

      {error && (
        <div className="bg-red-500/20 border border-red-500 text-red-500 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Total Networks</p>
          <p className="text-2xl font-bold">{blockchains.length}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">EVM Chains</p>
          <p className="text-2xl font-bold">{evmChains}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Non-EVM Chains</p>
          <p className="text-2xl font-bold">{nonEvmChains}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Active</p>
          <p className="text-2xl font-bold text-green-500">{activeChains}</p>
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        {['Networks', 'RPC', 'Explorers', 'Nodes'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab.toLowerCase())} className={`px-4 py-2 rounded-lg ${activeTab === tab.toLowerCase() ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
            {tab}
          </button>
        ))}
      </div>

      {activeTab === 'networks' && (
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Blockchain</th>
                <th className="p-3 text-left">Symbol</th>
                <th className="p-3 text-left">Chain ID</th>
                <th className="p-3 text-left">Type</th>
                <th className="p-3 text-left">Status</th>
                <th className="p-3 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {blockchains.map(chain => (
                <tr key={chain.id} className="border-t border-slate-700">
                  <td className="p-3 font-bold">{chain.name}</td>
                  <td className="p-3 text-amber-500">{chain.symbol}</td>
                  <td className="p-3">{chain.chainId || 'N/A'}</td>
                  <td className="p-3">{chain.type}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${chain.status === 'Active' ? 'bg-green-500/20 text-green-500' : 'bg-red-500/20 text-red-500'}`}>
                      {chain.status}
                    </span>
                  </td>
                  <td className="p-3">
                    <div className="flex gap-2">
                      <button className="btn btn-secondary text-xs">Edit</button>
                      <button className="btn btn-danger text-xs">{chain.status === 'Active' ? 'Disable' : 'Enable'}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'rpc' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">RPC Endpoints</h3>
          <p className="text-sm opacity-60 mb-4">Manage RPC endpoints for each blockchain:</p>
          <div className="space-y-3">
            {blockchains.map(chain => (
              <div key={chain.id} className="flex justify-between items-center p-3 bg-slate-700 rounded-lg">
                <div>
                  <p className="font-semibold">{chain.name}</p>
                  <p className="text-xs opacity-60 font-mono">{chain.rpc}</p>
                </div>
                <button className="btn btn-secondary text-xs">Test</button>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'explorers' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Block Explorers</h3>
          <div className="space-y-3">
            {blockchains.map(chain => (
              <div key={chain.id} className="flex justify-between items-center p-3 bg-slate-700 rounded-lg">
                <div>
                  <p className="font-semibold">{chain.name}</p>
                  <p className="text-xs opacity-60 font-mono">{chain.explorer}</p>
                </div>
                <button className="btn btn-secondary text-xs">Open</button>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'nodes' && (
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Node Infrastructure</h3>
          <p className="text-sm opacity-60">Node management and monitoring coming soon...</p>
        </div>
      )}
    </div>
  );
}

export default BlockchainManagement;
