// TigerWallet Admin - Token Management Page
// Manage platform tokens

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface Token {
  id: string;
  address: string;
  name: string;
  symbol: string;
  decimals: number;
  isListed: boolean;
  isPaused: boolean;
  chain: string;
  price?: string;
  volume24h?: string;
  marketCap?: string;
}

const TokensPage: React.FC = () => {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [listedFilter, setListedFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [showAddModal, setShowAddModal] = useState(false);
  const [newToken, setNewToken] = useState({
    address: '',
    name: '',
    symbol: '',
    decimals: 18,
    chain: 'ethereum',
  });

  useEffect(() => {
    loadTokens();
  }, [page, chainFilter, listedFilter]);

  const loadTokens = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await adminApi.getTokens({
        page,
        pageSize: 20,
        search: searchTerm || undefined,
        chain: chainFilter || undefined,
        listed: listedFilter === 'true' ? true : listedFilter === 'false' ? false : undefined,
      });

      setTokens(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tokens');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadTokens();
  };

  const handleListToken = async (address: string) => {
    try {
      await adminApi.listToken(address);
      loadTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to list token');
    }
  };

  const handleDelistToken = async (address: string) => {
    try {
      await adminApi.delistToken(address);
      loadTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delist token');
    }
  };

  const handlePauseToken = async (address: string) => {
    try {
      await adminApi.pauseToken(address);
      loadTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to pause token');
    }
  };

  const handleUnpauseToken = async (address: string) => {
    try {
      await adminApi.unpauseToken(address);
      loadTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unpause token');
    }
  };

  const handleAddToken = async () => {
    try {
      await adminApi.createToken(newToken);
      setShowAddModal(false);
      setNewToken({ address: '', name: '', symbol: '', decimals: 18, chain: 'ethereum' });
      loadTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add token');
    }
  };

  const truncateAddress = (address: string): string => {
    return address.slice(0, 6) + '...' + address.slice(-4);
  };

  const formatCurrency = (value?: string): string => {
    if (!value) return '-';
    const num = parseFloat(value);
    if (isNaN(num)) return '-';
    return '$' + num.toLocaleString();
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
            Token Management
          </h1>
          <p style={{ color: 'var(--text-secondary)' }}>
            Manage platform tokens and listings
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowAddModal(true)}>
          + Add Token
        </button>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {/* Search and Filters */}
      <div className="card mb-6">
        <div className="card-body">
          <form onSubmit={handleSearch} className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-64">
              <label className="form-label">Search</label>
              <input
                type="text"
                className="form-input"
                placeholder="Search by name or symbol..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <div className="w-40">
              <label className="form-label">Chain</label>
              <select
                className="form-select"
                value={chainFilter}
                onChange={(e) => setChainFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="ethereum">Ethereum</option>
                <option value="bsc">BNB Chain</option>
                <option value="polygon">Polygon</option>
                <option value="arbitrum">Arbitrum</option>
              </select>
            </div>
            <div className="w-40">
              <label className="form-label">Listed</label>
              <select
                className="form-select"
                value={listedFilter}
                onChange={(e) => setListedFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="true">Listed</option>
                <option value="false">Unlisted</option>
              </select>
            </div>
            <button type="submit" className="btn btn-primary">
              Search
            </button>
          </form>
        </div>
      </div>

      {/* Tokens Table */}
      <div className="card">
        <div className="card-body p-0">
          {loading ? (
            <div className="flex items-center justify-center p-8">
              <div className="loader"></div>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="table">
                <thead>
                  <tr>
                    <th>Token</th>
                    <th>Address</th>
                    <th>Chain</th>
                    <th>Price</th>
                    <th>24h Volume</th>
                    <th>Market Cap</th>
                    <th>Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {tokens.length > 0 ? (
                    tokens.map((token) => (
                      <tr key={token.id}>
                        <td>
                          <div className="flex items-center gap-2">
                            <div className="w-8 h-8 rounded-full flex items-center justify-center" 
                              style={{ backgroundColor: 'var(--color-primary)', color: 'white', fontWeight: 600 }}>
                              {token.symbol.charAt(0)}
                            </div>
                            <div>
                              <p className="font-medium" style={{ color: 'var(--text-primary)' }}>
                                {token.name}
                              </p>
                              <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                                {token.symbol}
                              </p>
                            </div>
                          </div>
                        </td>
                        <td>
                          <code className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                            {truncateAddress(token.address)}
                          </code>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>{token.chain}</span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-primary)' }}>
                            {formatCurrency(token.price)}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {formatCurrency(token.volume24h)}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {formatCurrency(token.marketCap)}
                          </span>
                        </td>
                        <td>
                          <div className="flex gap-1">
                            <span className={`badge ${token.isListed ? 'badge-success' : 'badge-neutral'}`}>
                              {token.isListed ? 'Listed' : 'Unlisted'}
                            </span>
                            {token.isPaused && (
                              <span className="badge badge-warning">Paused</span>
                            )}
                          </div>
                        </td>
                        <td>
                          <div className="flex gap-1 flex-wrap">
                            {token.isListed ? (
                              <button
                                className="btn btn-sm btn-outline"
                                onClick={() => handleDelistToken(token.address)}
                              >
                                Delist
                              </button>
                            ) : (
                              <button
                                className="btn btn-sm btn-success"
                                onClick={() => handleListToken(token.address)}
                              >
                                List
                              </button>
                            )}
                            {token.isPaused ? (
                              <button
                                className="btn btn-sm btn-secondary"
                                onClick={() => handleUnpauseToken(token.address)}
                              >
                                Unpause
                              </button>
                            ) : (
                              <button
                                className="btn btn-sm btn-warning"
                                onClick={() => handlePauseToken(token.address)}
                              >
                                Pause
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={8} className="text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
                        No tokens found
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="card-footer flex justify-between items-center">
            <span style={{ color: 'var(--text-secondary)' }}>
              Page {page} of {totalPages}
            </span>
            <div className="pagination">
              <button
                className="pagination-btn"
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                Previous
              </button>
              <button
                className="pagination-btn"
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Add Token Modal */}
      {showAddModal && (
        <div className="modal-overlay" onClick={() => setShowAddModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Add New Token</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowAddModal(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label className="form-label">Token Address</label>
                <input
                  type="text"
                  className="form-input"
                  value={newToken.address}
                  onChange={(e) => setNewToken({ ...newToken, address: e.target.value })}
                  placeholder="0x..."
                />
              </div>
              <div className="form-group">
                <label className="form-label">Name</label>
                <input
                  type="text"
                  className="form-input"
                  value={newToken.name}
                  onChange={(e) => setNewToken({ ...newToken, name: e.target.value })}
                  placeholder="Token Name"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Symbol</label>
                <input
                  type="text"
                  className="form-input"
                  value={newToken.symbol}
                  onChange={(e) => setNewToken({ ...newToken, symbol: e.target.value })}
                  placeholder="SYMBOL"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Decimals</label>
                <input
                  type="number"
                  className="form-input"
                  value={newToken.decimals}
                  onChange={(e) => setNewToken({ ...newToken, decimals: parseInt(e.target.value) })}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Chain</label>
                <select
                  className="form-select"
                  value={newToken.chain}
                  onChange={(e) => setNewToken({ ...newToken, chain: e.target.value })}
                >
                  <option value="ethereum">Ethereum</option>
                  <option value="bsc">BNB Chain</option>
                  <option value="polygon">Polygon</option>
                  <option value="arbitrum">Arbitrum</option>
                </select>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowAddModal(false)}>
                Cancel
              </button>
              <button className="btn btn-primary" onClick={handleAddToken}>
                Add Token
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TokensPage;
