import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { pairService } from '../services/api';

interface Pair {
  id: string;
  pair_id: string;
  base_symbol: string;
  quote_symbol: string;
  pair_name: string;
  chain_id: string;
  status: string;
  trading_enabled: boolean;
  maker_fee: number;
  taker_fee: number;
  current_price: number | null;
  price_change_24h: number | null;
  volume_24h: number | null;
  created_at: string;
}

export default function Pairs() {
  const { isDark } = useTheme();
  const [pairs, setPairs] = useState<Pair[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedPair, setSelectedPair] = useState<Pair | null>(null);

  useEffect(() => {
    loadPairs();
  }, [page, statusFilter, chainFilter]);

  const loadPairs = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await pairService.getPairs({
        page,
        limit: 20,
        status: statusFilter || undefined,
        chain: chainFilter || undefined
      });
      
      setPairs(response.data);
      setTotalPages(response.meta.total_pages);
    } catch (err: any) {
      setError(err.message || 'Failed to load pairs');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleTrading = async (pairId: string, enabled: boolean) => {
    try {
      await pairService.updatePair(pairId, { trading_enabled: enabled });
      alert(`Trading ${enabled ? 'enabled' : 'disabled'} successfully`);
      loadPairs();
    } catch (err: any) {
      alert(err.message || 'Failed to update pair');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800';
      case 'inactive': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Trading Pairs
          </h1>
          <button
            onClick={() => loadPairs()}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
          >
            Refresh
          </button>
        </div>

        {/* Filters */}
        <div className={`rounded-lg shadow p-4 mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="flex flex-wrap gap-4">
            <select
              value={statusFilter}
              onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Status</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
            <select
              value={chainFilter}
              onChange={(e) => { setChainFilter(e.target.value); setPage(1); }}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Chains</option>
              <option value="1">Ethereum</option>
              <option value="56">BSC</option>
              <option value="137">Polygon</option>
            </select>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {/* Pairs Table */}
        <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          {loading ? (
            <div className="p-8 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent"></div>
              <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Loading...</p>
            </div>
          ) : pairs.length === 0 ? (
            <div className="p-8 text-center">
              <p className={`${isDark ? 'text-gray-400' : 'text-gray-600'}`}>No pairs found</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                  <tr>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Pair</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Price</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>24h Change</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Volume</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Fees</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
                  </tr>
                </thead>
                <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                  {pairs.map((pair) => (
                    <tr key={pair.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        <div>
                          <p className="font-medium">{pair.base_symbol}/{pair.quote_symbol}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{pair.pair_name}</p>
                        </div>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {pair.chain_id}
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(pair.status)}`}>
                          {pair.status}
                        </span>
                      </td>
                      <td className={`px-4 py-4 font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {pair.current_price ? `$${pair.current_price.toLocaleString()}` : 'N/A'}
                      </td>
                      <td className={`px-4 py-4 ${pair.price_change_24h && pair.price_change_24h >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                        {pair.price_change_24h !== null ? `${pair.price_change_24h >= 0 ? '+' : ''}${pair.price_change_24h.toFixed(2)}%` : 'N/A'}
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {pair.volume_24h ? `$${pair.volume_24h.toLocaleString()}` : 'N/A'}
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        M: {pair.maker_fee}% / T: {pair.taker_fee}%
                      </td>
                      <td className="px-4 py-4">
                        <button
                          onClick={() => handleToggleTrading(pair.id, !pair.trading_enabled)}
                          className={`px-3 py-1 text-sm rounded ${pair.trading_enabled ? (isDark ? 'bg-red-600 hover:bg-red-700' : 'bg-red-500 hover:bg-red-600') : (isDark ? 'bg-green-600 hover:bg-green-700' : 'bg-green-500 hover:bg-green-600')} text-white`}
                        >
                          {pair.trading_enabled ? 'Disable' : 'Enable'}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {totalPages > 1 && (
          <div className="flex justify-center items-center gap-2 mt-6">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-white hover:bg-gray-50'} ${page === 1 ? 'opacity-50' : ''} ${isDark ? 'text-white' : 'text-gray-700'}`}
            >
              Previous
            </button>
            <span className={`px-4 py-2 ${isDark ? 'text-white' : 'text-gray-700'}`}>
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-white hover:bg-gray-50'} ${page === totalPages ? 'opacity-50' : ''} ${isDark ? 'text-white' : 'text-gray-700'}`}
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
