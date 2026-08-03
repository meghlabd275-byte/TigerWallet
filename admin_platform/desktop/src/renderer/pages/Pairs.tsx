import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { pairService } from '../services/api';

export default function Pairs() {
  const { isDark } = useTheme();
  const [pairs, setPairs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadPairs(); }, []);

  const loadPairs = async () => {
    setLoading(true);
    try {
      const response = await pairService.getPairs({ page: 1, limit: 50 });
      setPairs(response.data);
    } catch (err) { console.error('Failed:', err); }
    finally { setLoading(false); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Trading Pairs</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : pairs.length === 0 ? <div className="p-8 text-center">No pairs</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Pair</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Price</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Fees</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {pairs.map((pair) => (
                <tr key={pair.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{pair.base_symbol}/{pair.quote_symbol}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{pair.chain_id}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${pair.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>{pair.status}</span></td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{pair.current_price ? `$${pair.current_price.toLocaleString()}` : 'N/A'}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>M: {pair.maker_fee}% / T: {pair.taker_fee}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
