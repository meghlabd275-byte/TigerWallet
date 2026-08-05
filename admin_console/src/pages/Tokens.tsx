/**
 * Tokens - Admin Console
 */
import React, { useEffect, useState } from 'react';
import { adminConsoleApi } from '../services/api';

export default function Tokens() {
  const [tokens, setTokens] = useState<any[]>([]);
  useEffect(() => { adminConsoleApi.getTokens().then(d => setTokens(d.data || [])).catch(console.error); }, []);
  const handleList = async (addr: string) => { await adminConsoleApi.listToken(addr); window.location.reload(); };
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Token Management</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700"><tr><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Symbol</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th></tr></thead>
          <tbody className="divide-y divide-gray-200">
            {tokens.map(t => (<tr key={t.address}><td className="px-6 py-4">{t.name}</td><td className="px-6 py-4">{t.symbol}</td><td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${t.isListed ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>{t.isListed ? 'Listed' : 'Unlisted'}</span></td><td className="px-6 py-4">{!t.isListed && <button onClick={() => handleList(t.address)} className="text-blue-600 hover:underline">List</button>}</td></tr>))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
