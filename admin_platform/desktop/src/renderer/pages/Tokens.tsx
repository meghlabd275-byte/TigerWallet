import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { tokenService } from '../services/api';

interface Token {
  id: string;
  name: string;
  symbol: string;
  contract_addr: string;
  chain_id: string;
  chain_name: string;
  is_active: boolean;
  is_verified: boolean;
  price: number | null;
  price_change_24h: number | null;
}

export default function Tokens() {
  const { isDark } = useTheme();
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadTokens(); }, []);

  const loadTokens = async () => {
    setLoading(true);
    try {
      const response = await tokenService.getTokens({ page: 1, limit: 50 });
      setTokens(response.data);
    } catch (err) { console.error('Failed to load tokens:', err); }
    finally { setLoading(false); }
  };

  const handleVerify = async (id: string) => {
    try { await tokenService.verifyToken(id); loadTokens(); } catch (err) { alert('Failed to verify'); }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete token?')) return;
    try { await tokenService.deleteToken(id); loadTokens(); } catch (err) { alert('Failed to delete'); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Tokens</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : tokens.length === 0 ? <div className="p-8 text-center">No tokens</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Token</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Price</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>24h</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {tokens.map((token) => (
                <tr key={token.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                    <div><p className="font-medium">{token.name}</p><p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{token.symbol}</p></div>
                  </td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{token.chain_name}</td>
                  <td className="px-4 py-4">
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${token.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>{token.is_active ? 'Active' : 'Inactive'}</span>
                    {token.is_verified && <span className="ml-2 text-green-500">✓</span>}
                  </td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{token.price ? `$${token.price.toLocaleString()}` : 'N/A'}</td>
                  <td className={`px-4 py-4 ${(token.price_change_24h ?? 0) >= 0 ? 'text-green-500' : 'text-red-500'}`}>{token.price_change_24h !== null ? `${token.price_change_24h >= 0 ? '+' : ''}${token.price_change_24h.toFixed(2)}%` : 'N/A'}</td>
                  <td className="px-4 py-4">
                    <div className="flex gap-2">
                      {!token.is_verified && <button onClick={() => handleVerify(token.id)} className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-yellow-600' : 'bg-yellow-500'} text-white`}>Verify</button>}
                      <button onClick={() => handleDelete(token.id)} className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-red-600' : 'bg-red-500'} text-white`}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
