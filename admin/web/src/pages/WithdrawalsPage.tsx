// TigerWallet Admin - Withdrawals Page
// Manage withdrawal requests

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface Withdrawal {
  id: string;
  userId: string;
  userEmail: string;
  amount: string;
  token: string;
  chain: string;
  toAddress: string;
  status: string;
  fee: string;
  txHash?: string;
  requestedAt: string;
  processedAt?: string;
  rejectionReason?: string;
}

const WithdrawalsPage: React.FC = () => {
  const [withdrawals, setWithdrawals] = useState<Withdrawal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedWithdrawal, setSelectedWithdrawal] = useState<Withdrawal | null>(null);
  const [showRejectModal, setShowRejectModal] = useState(false);
  const [rejectReason, setRejectReason] = useState('');
  const [processing, setProcessing] = useState<string | null>(null);

  useEffect(() => {
    loadWithdrawals();
  }, [page, statusFilter]);

  const loadWithdrawals = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await adminApi.getWithdrawals({
        page,
        pageSize: 20,
        status: statusFilter || undefined,
      });

      setWithdrawals(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load withdrawals');
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      setProcessing(id);
      await adminApi.approveWithdrawal(id);
      loadWithdrawals();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve withdrawal');
    } finally {
      setProcessing(null);
    }
  };

  const handleReject = async () => {
    if (!selectedWithdrawal || !rejectReason) return;
    
    try {
      await adminApi.rejectWithdrawal(selectedWithdrawal.id, rejectReason);
      setShowRejectModal(false);
      setRejectReason('');
      loadWithdrawals();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject withdrawal');
    }
  };

  const handleBatchApprove = async (ids: string[]) => {
    try {
      setProcessing('batch');
      await adminApi.batchApproveWithdrawals(ids);
      loadWithdrawals();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to batch approve');
    } finally {
      setProcessing(null);
    }
  };

  const getStatusBadgeClass = (status: string): string => {
    switch (status) {
      case 'approved': case 'completed': return 'badge-success';
      case 'pending': case 'processing': return 'badge-warning';
      case 'rejected': case 'failed': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  const truncateAddress = (address: string): string => {
    return address.slice(0, 6) + '...' + address.slice(-4);
  };

  const pendingIds = withdrawals.filter(w => w.status === 'pending').map(w => w.id);

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
            Withdrawal Management
          </h1>
          <p style={{ color: 'var(--text-secondary)' }}>
            Review and process withdrawal requests
          </p>
        </div>
        {pendingIds.length > 0 && (
          <button
            className="btn btn-primary"
            onClick={() => handleBatchApprove(pendingIds)}
            disabled={processing === 'batch'}
          >
            {processing === 'batch' ? 'Processing...' : `Approve All (${pendingIds.length})`}
          </button>
        )}
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="stat-card">
          <div className="stat-label">Pending</div>
          <div className="stat-value">
            {withdrawals.filter(w => w.status === 'pending').length}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Processing</div>
          <div className="stat-value">
            {withdrawals.filter(w => w.status === 'processing').length}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Completed</div>
          <div className="stat-value">
            {withdrawals.filter(w => w.status === 'completed').length}
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Rejected</div>
          <div className="stat-value">
            {withdrawals.filter(w => w.status === 'rejected').length}
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="card mb-6">
        <div className="card-body">
          <div className="flex gap-4 items-end">
            <div className="w-40">
              <label className="form-label">Status</label>
              <select
                className="form-select"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="processing">Processing</option>
                <option value="approved">Approved</option>
                <option value="rejected">Rejected</option>
                <option value="completed">Completed</option>
                <option value="failed">Failed</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* Withdrawals Table */}
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
                    <th>User</th>
                    <th>Amount</th>
                    <th>Token</th>
                    <th>Chain</th>
                    <th>To Address</th>
                    <th>Fee</th>
                    <th>Status</th>
                    <th>Requested</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {withdrawals.length > 0 ? (
                    withdrawals.map((withdrawal) => (
                      <tr key={withdrawal.id}>
                        <td>
                          <span style={{ color: 'var(--text-primary)' }}>
                            {withdrawal.userEmail}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                            {withdrawal.amount}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {withdrawal.token}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {withdrawal.chain}
                          </span>
                        </td>
                        <td>
                          <code className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                            {truncateAddress(withdrawal.toAddress)}
                          </code>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {withdrawal.fee}
                          </span>
                        </td>
                        <td>
                          <span className={`badge ${getStatusBadgeClass(withdrawal.status)}`}>
                            {withdrawal.status}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-tertiary)' }}>
                            {new Date(withdrawal.requestedAt).toLocaleString()}
                          </span>
                        </td>
                        <td>
                          <div className="flex gap-1">
                            {withdrawal.status === 'pending' && (
                              <>
                                <button
                                  className="btn btn-sm btn-success"
                                  onClick={() => handleApprove(withdrawal.id)}
                                  disabled={processing === withdrawal.id}
                                >
                                  {processing === withdrawal.id ? '...' : 'Approve'}
                                </button>
                                <button
                                  className="btn btn-sm btn-danger"
                                  onClick={() => {
                                    setSelectedWithdrawal(withdrawal);
                                    setShowRejectModal(true);
                                  }}
                                >
                                  Reject
                                </button>
                              </>
                            )}
                            {withdrawal.txHash && (
                              <a
                                href={`https://etherscan.io/tx/${withdrawal.txHash}`}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="btn btn-sm btn-outline"
                              >
                                View
                              </a>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={9} className="text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
                        No withdrawals found
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

      {/* Reject Modal */}
      {showRejectModal && selectedWithdrawal && (
        <div className="modal-overlay" onClick={() => setShowRejectModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Reject Withdrawal</h3>
              <button className="btn btn-sm btn-outline" onClick={() => setShowRejectModal(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
                Reject withdrawal of <strong>{selectedWithdrawal.amount} {selectedWithdrawal.token}</strong> from <strong>{selectedWithdrawal.userEmail}</strong>
              </p>
              <div className="form-group">
                <label className="form-label">Rejection Reason</label>
                <textarea
                  className="form-textarea"
                  rows={4}
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                  placeholder="Enter rejection reason..."
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowRejectModal(false)}>
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={handleReject}
                disabled={!rejectReason}
              >
                Reject
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WithdrawalsPage;
