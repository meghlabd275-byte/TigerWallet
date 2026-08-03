import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { chainService } from '../services/api';

interface Chain {
  id: string;
  chain_id: number;
  name: string;
  symbol: string;
  type: string;
  rpc_urls: string[];
  explorer_urls: string[];
  is_active: boolean;
  is_testnet: boolean;
  confirmations: number;
  created_at: string;
}

export default function Chains() {
  const { isDark } = useTheme();
  const [chains, setChains] = useState<Chain[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedChain, setSelectedChain] = useState<Chain | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);

  useEffect(() => {
    loadChains();
  }, []);

  const loadChains = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await chainService.getChains();
      setChains(response.data);
    } catch (err: any) {
      setError(err.message || 'Failed to load chains');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleActive = async (chainId: string, currentStatus: boolean) => {
    try {
      await chainService.updateChain(chainId, { is_active: !currentStatus });
      alert(`Chain ${!currentStatus ? 'activated' : 'deactivated'} successfully`);
      loadChains();
    } catch (err: any) {
      alert(err.message || 'Failed to update chain');
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Blockchain Chains
          </h1>
          <div className="flex gap-2">
            <button
              onClick={() => loadChains()}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-white hover:bg-gray-50'} ${isDark ? 'text-white' : 'text-gray-700'} border`}
            >
              Refresh
            </button>
            <button
              onClick={() => setShowAddModal(true)}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
            >
              Add Chain
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          {loading ? (
            <div className="p-8 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent"></div>
              <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Loading...</p>
            </div>
          ) : chains.length === 0 ? (
            <div className="p-8 text-center">
              <p className={`${isDark ? 'text-gray-400' : 'text-gray-600'}`}>No chains found</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                  <tr>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain ID</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Type</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Testnet</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Confirmations</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
                  </tr>
                </thead>
                <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                  {chains.map((chain) => (
                    <tr key={chain.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        <div>
                          <p className="font-medium">{chain.name}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{chain.symbol}</p>
                        </div>
                      </td>
                      <td className={`px-4 py-4 font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {chain.chain_id}
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {chain.type}
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${chain.is_testnet ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'}`}>
                          {chain.is_testnet ? 'Yes' : 'No'}
                        </span>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {chain.confirmations}
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${chain.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                          {chain.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex gap-2">
                          <button
                            onClick={() => setSelectedChain(chain)}
                            className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
                          >
                            View
                          </button>
                          <button
                            onClick={() => handleToggleActive(chain.id, chain.is_active)}
                            className={`px-3 py-1 text-sm rounded ${chain.is_active ? (isDark ? 'bg-red-600 hover:bg-red-700' : 'bg-red-500 hover:bg-red-600') : (isDark ? 'bg-green-600 hover:bg-green-700' : 'bg-green-500 hover:bg-green-600')} text-white`}
                          >
                            {chain.is_active ? 'Disable' : 'Enable'}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {selectedChain && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  {selectedChain.name}
                </h2>
                <button
                  onClick={() => setSelectedChain(null)}
                  className={`text-2xl ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-700'}`}
                >
                  ×
                </button>
              </div>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Chain ID</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedChain.chain_id}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Symbol</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedChain.symbol}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Type</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedChain.type}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Confirmations</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedChain.confirmations}</p>
                  </div>
                  <div className="col-span-2">
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>RPC URLs</label>
                    <div className="mt-1 space-y-1">
                      {selectedChain.rpc_urls.map((url, i) => (
                        <p key={i} className={`text-sm font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>{url}</p>
                      ))}
                    </div>
                  </div>
                  <div className="col-span-2">
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Explorer URLs</label>
                    <div className="mt-1 space-y-1">
                      {selectedChain.explorer_urls.map((url, i) => (
                        <p key={i} className={`text-sm font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>{url}</p>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
