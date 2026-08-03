import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { tokenService } from '../services/api';

interface Token {
  id: string;
  token_id: string;
  name: string;
  symbol: string;
  contract_addr: string;
  decimals: number;
  chain_id: string;
  chain_name: string;
  is_active: boolean;
  is_verified: boolean;
  is_native_token: boolean;
  logo_url: string | null;
  website: string | null;
  price: number | null;
  price_change_24h: number | null;
  volume_24h: number | null;
  created_at: string;
}

export default function Tokens() {
  const { isDark } = useTheme();
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedToken, setSelectedToken] = useState<Token | null>(null);

  useEffect(() => {
    loadTokens();
  }, [page, statusFilter, chainFilter]);

  const loadTokens = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await tokenService.getTokens({
        page,
        limit: 20,
        status: statusFilter || undefined,
        chain: chainFilter || undefined,
        search: search || undefined
      });
      
      setTokens(response.data);
      setTotalPages(response.meta.total_pages);
    } catch (err: any) {
      setError(err.message || 'Failed to load tokens');
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (tokenId: string) => {
    try {
      await tokenService.verifyToken(tokenId);
      alert('Token verified successfully');
      loadTokens();
    } catch (err: any) {
      alert(err.message || 'Failed to verify token');
    }
  };

  const handleDelete = async (tokenId: string) => {
    if (!confirm('Are you sure you want to delete this token?')) return;
    
    try {
      await tokenService.deleteToken(tokenId);
      alert('Token deleted successfully');
      loadTokens();
    } catch (err: any) {
      alert(err.message || 'Failed to delete token');
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Tokens
          </h1>
          <div className="flex gap-2">
            <button
              onClick={() => loadTokens()}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-white hover:bg-gray-50'} ${isDark ? 'text-white' : 'text-gray-700'} border`}
            >
              Refresh
            </button>
            <button
              onClick={() => setShowCreateModal(true)}
              className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
            >
              Add Token
            </button>
          </div>
        </div>

        {/* Filters */}
        <div className={`rounded-lg shadow p-4 mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <div className="flex flex-wrap gap-4">
            <input
              type="text"
              placeholder="Search tokens..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && loadTokens()}
              className={`flex-1 min-w-[200px] px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            />
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

        {/* Error Message */}
        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {/* Tokens Grid */}
        {loading ? (
          <div className={`rounded-lg shadow p-8 text-center ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent"></div>
            <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Loading...</p>
          </div>
        ) : tokens.length === 0 ? (
          <div className={`rounded-lg shadow p-8 text-center ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <p className={`${isDark ? 'text-gray-400' : 'text-gray-600'}`}>No tokens found</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {tokens.map((token) => (
              <div
                key={token.id}
                className={`rounded-lg shadow p-6 ${isDark ? 'bg-gray-800' : 'bg-white'} ${!token.is_active ? 'opacity-60' : ''}`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    {token.logo_url ? (
                      <img src={token.logo_url} alt={token.name} className="w-10 h-10 rounded-full" />
                    ) : (
                      <div className="w-10 h-10 rounded-full bg-gray-300 flex items-center justify-center">
                        <span className="text-xl font-bold">{token.symbol.charAt(0)}</span>
                      </div>
                    )}
                    <div>
                      <h3 className={`font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {token.name}
                      </h3>
                      <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {token.symbol}
                      </p>
                    </div>
                  </div>
                  <div className="flex gap-1">
                    {token.is_verified ? (
                      <span className="text-green-500 text-lg" title="Verified">✓</span>
                    ) : (
                      <button
                        onClick={() => handleVerify(token.id)}
                        className={`text-xs px-2 py-1 rounded ${isDark ? 'bg-yellow-600' : 'bg-yellow-500'} text-white`}
                      >
                        Verify
                      </button>
                    )}
                  </div>
                </div>

                <div className="mt-4 space-y-2">
                  <div className="flex justify-between">
                    <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Status</span>
                    <span className={`px-2 py-0.5 text-xs rounded-full ${token.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                      {token.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Chain</span>
                    <span className={`text-sm ${isDark ? 'text-white' : 'text-gray-900'}`}>{token.chain_name}</span>
                  </div>
                  {token.price && (
                    <div className="flex justify-between">
                      <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Price</span>
                      <span className={`text-sm font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        ${token.price.toLocaleString()}
                      </span>
                    </div>
                  )}
                  {token.price_change_24h !== null && (
                    <div className="flex justify-between">
                      <span className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>24h Change</span>
                      <span className={`text-sm font-medium ${token.price_change_24h >= 0 ? 'text-green-500' : 'text-red-500'}`}>
                        {token.price_change_24h >= 0 ? '+' : ''}{token.price_change_24h.toFixed(2)}%
                      </span>
                    </div>
                  )}
                </div>

                <div className="mt-4 flex gap-2">
                  <button
                    onClick={() => setSelectedToken(token)}
                    className={`flex-1 py-2 rounded text-sm ${isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-100 hover:bg-gray-200'} ${isDark ? 'text-white' : 'text-gray-700'}`}
                  >
                    View Details
                  </button>
                  <button
                    onClick={() => handleDelete(token.id)}
                    className="px-3 py-2 rounded text-sm bg-red-100 text-red-700 hover:bg-red-200"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
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

        {/* Token Detail Modal */}
        {selectedToken && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  Token Details
                </h2>
                <button
                  onClick={() => setSelectedToken(null)}
                  className={`text-2xl ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-700'}`}
                >
                  ×
                </button>
              </div>
              
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Name</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedToken.name}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Symbol</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedToken.symbol}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Contract Address</label>
                    <p className={`text-lg font-mono ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedToken.contract_addr}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Decimals</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedToken.decimals}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Chain</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedToken.chain_name}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Verified</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedToken.is_verified ? 'Yes' : 'No'}</p>
                  </div>
                  {selectedToken.website && (
                    <div>
                      <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Website</label>
                      <a href={selectedToken.website} target="_blank" rel="noopener noreferrer" className="text-blue-500 hover:underline">
                        {selectedToken.website}
                      </a>
                    </div>
                  )}
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Created At</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{new Date(selectedToken.created_at).toLocaleString()}</p>
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
