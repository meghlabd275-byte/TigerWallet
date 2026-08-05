/**
 * Tokens Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';

export default function Tokens() {
  const [tokens, setTokens] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadTokens(); }, []);

  const loadTokens = async () => {
    try {
      const data = await whiteLabelAdminApi.getTokens();
      setTokens(data.data || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleList = async (address: string) => {
    try {
      await whiteLabelAdminApi.listToken(address);
      loadTokens();
    } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Token Management</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Symbol</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Address</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {tokens.map((token) => (
              <tr key={token.address}>
                <td className="px-6 py-4">{token.name}</td>
                <td className="px-6 py-4">{token.symbol}</td>
                <td className="px-6 py-4 font-mono text-sm">{token.address?.substring(0, 10)}...</td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${token.isListed ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>{token.isListed ? 'Listed' : 'Unlisted'}</span></td>
                <td className="px-6 py-4">
                  {!token.isListed && <button onClick={() => handleList(token.address)} className="text-blue-600 hover:underline">List</button>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
