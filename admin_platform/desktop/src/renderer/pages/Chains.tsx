import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { chainService } from '../services/api';

export default function Chains() {
  const { isDark } = useTheme();
  const [chains, setChains] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadChains(); }, []);

  const loadChains = async () => {
    setLoading(true);
    try {
      const response = await chainService.getChains();
      setChains(response.data);
    } catch (err) { console.error('Failed:', err); }
    finally { setLoading(false); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Blockchain Chains</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : chains.length === 0 ? <div className="p-8 text-center">No chains</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain ID</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Type</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Testnet</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {chains.map((chain) => (
                <tr key={chain.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}><div><p className="font-medium">{chain.name}</p><p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{chain.symbol}</p></div></td>
                  <td className={`px-4 py-4 font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>{chain.chain_id}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{chain.type}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${chain.is_testnet ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'}`}>{chain.is_testnet ? 'Yes' : 'No'}</span></td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${chain.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>{chain.is_active ? 'Active' : 'Inactive'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
