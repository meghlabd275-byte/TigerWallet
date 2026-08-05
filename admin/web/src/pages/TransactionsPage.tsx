// TigerWallet Admin - Transactions Page
// Transaction monitoring, flagging, and management

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  tokenSymbol: string;
  chain: string;
  status: string;
  type: string;
  timestamp: string;
  flagReason?: string;
}

const TransactionsPage: React.FC = () => {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedTx, setSelectedTx] = useState<Transaction | null>(null);
  const [showFlagModal, setShowFlagModal] = useState(false);
  const [flagReason, setFlagReason] = useState('');

  useEffect(() => {
    loadTransactions();
  }, [page, statusFilter, typeFilter, chainFilter]);

  const loadTransactions = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await adminApi.getTransactions({
        page,
        pageSize: 20,
        search: searchTerm || undefined,
        status: statusFilter || undefined,
        type: typeFilter || undefined,
        chain: chainFilter || undefined,
      });

      setTransactions(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load transactions');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadTransactions();
  };

  const handleFlagTransaction = async () => {
    if (!selectedTx || !flagReason) return;
    
    try {
      await adminApi.flagTransaction(selectedTx.id, flagReason);
      setShowFlagModal(false);
      setFlagReason('');
      loadTransactions();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to flag transaction');
    }
  };

  const handleUnflagTransaction = async (txId: string) => {
    try {
      await adminApi.unflagTransaction(txId);
      loadTransactions();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unflag transaction');
    }
  };

  const handleCancelTransaction = async (txId: string) => {
    if (!confirm('Are you sure you want to cancel this transaction?')) return;
    
    try {
      await adminApi.cancelTransaction(txId);
      loadTransactions();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel transaction');
    }
  };

  const getStatusBadgeClass = (status: string): string => {
    switch (status) {
      case 'confirmed': case 'completed': return 'badge-success';
      case 'pending': return 'badge-warning';
      case 'failed': case 'cancelled': return 'badge-error';
      case 'flagged': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  const truncateAddress = (address: string): string => {
    if (!address) return '';
    return address.slice(0, 6) + '...' + address.slice(-4);
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
          Transaction Management
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Monitor, flag, and manage platform transactions
        </p>
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
                placeholder="Search by hash, from, to..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <div className="w-36">
              <label className="form-label">Status</label>
              <select
                className="form-select"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="confirmed">Confirmed</option>
                <option value="failed">Failed</option>
                <option value="flagged">Flagged</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </div>
            <div className="w-36">
              <label className="form-label">Type</label>
              <select
                className="form-select"
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="transfer">Transfer</option>
                <option value="swap">Swap</option>
                <option value="stake">Stake</option>
                <option value="bridge">Bridge</option>
              </select>
            </div>
            <div className="w-36">
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
            <button type="submit" className="btn btn-primary">
              Search
            </button>
          </form>
        </div>
      </div>

      {/* Transactions Table */}
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
                    <th>Hash</th>
                    <th>Type</th>
                    <th>From</th>
                    <th>To</th>
                    <th>Amount</th>
                    <th>Chain</th>
                    <th>Status</th>
                    <th>Time</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.length > 0 ? (
                    transactions.map((tx) => (
                      <tr key={tx.id}>
                        <td>
                          <span className="font-mono text-sm" style={{ color: 'var(--text-primary)' }}>
                            {truncateAddress(tx.hash)}
                          </span>
                          {tx.flagReason && (
                            <span className="badge badge-error ml-2">Flagged</span>
                          )}
                        </td>
                        <td>
                          <span className="badge badge-neutral">{tx.type}</span>
                        </td>
                        <td>
                          <span className="font-mono text-sm" style={{ color: 'var(--text-secondary)' }}>
                            {truncateAddress(tx.from)}
                          </span>
                        </td>
                        <td>
                          <span className="font-mono text-sm" style={{ color: 'var(--text-secondary)' }}>
                            {truncateAddress(tx.to)}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                            {parseFloat(tx.amount).toFixed(4)} {tx.tokenSymbol}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>{tx.chain}</span>
                        </td>
                        <td>
                          <span className={`badge ${getStatusBadgeClass(tx.status)}`}>
                            {tx.status}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-tertiary)' }}>
                            {new Date(tx.timestamp).toLocaleString()}
                          </span>
                        </td>
                        <td>
                          <div className="flex gap-1">
                            {tx.flagReason ? (
                              <button
                                className="btn btn-sm btn-outline"
                                onClick={() => handleUnflagTransaction(tx.id)}
                              >
                                Unflag
                              </button>
                            ) : (
                              <button
                                className="btn btn-sm btn-outline"
                                onClick={() => {
                                  setSelectedTx(tx);
                                  setShowFlagModal(true);
                                }}
                              >
                                Flag
                              </button>
                            )}
                            {tx.status === 'pending' && (
                              <button
                                className="btn btn-sm btn-danger"
                                onClick={() => handleCancelTransaction(tx.id)}
                              >
                                Cancel
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={9} className="text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
                        No transactions found
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

      {/* Flag Modal */}
      {showFlagModal && selectedTx && (
        <div className="modal-overlay" onClick={() => setShowFlagModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Flag Transaction</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowFlagModal(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
                Transaction: <code>{selectedTx.hash.slice(0, 10)}...</code>
              </p>
              <div className="form-group">
                <label className="form-label">Reason for flagging</label>
                <textarea
                  className="form-textarea"
                  rows={4}
                  value={flagReason}
                  onChange={(e) => setFlagReason(e.target.value)}
                  placeholder="Enter reason for flagging..."
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowFlagModal(false)}>
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={handleFlagTransaction}
                disabled={!flagReason}
              >
                Flag Transaction
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TransactionsPage;
