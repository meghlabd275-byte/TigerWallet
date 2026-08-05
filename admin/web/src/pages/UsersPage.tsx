// TigerWallet Admin - Users Page
// User management with search, filters, and actions

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';

interface User {
  id: string;
  email: string;
  username: string;
  status: string;
  kycStatus: string;
  kycLevel: number;
  totalVolume: string;
  createdAt: string;
  lastLogin?: string;
  verified: boolean;
  suspended: boolean;
}

const UsersPage: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [kycFilter, setKycFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [suspendReason, setSuspendReason] = useState('');

  useEffect(() => {
    loadUsers();
  }, [page, statusFilter, kycFilter]);

  const loadUsers = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await adminApi.getUsers({
        page,
        pageSize: 20,
        search: searchTerm || undefined,
        status: statusFilter || undefined,
        kycStatus: kycFilter || undefined,
      });

      setUsers(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadUsers();
  };

  const handleSuspendUser = async () => {
    if (!selectedUser || !suspendReason) return;
    
    try {
      await adminApi.suspendUser(selectedUser.id, suspendReason);
      setShowModal(false);
      setSuspendReason('');
      loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to suspend user');
    }
  };

  const handleUnsuspendUser = async (userId: string) => {
    try {
      await adminApi.unsuspendUser(userId);
      loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unsuspend user');
    }
  };

  const handleVerifyUser = async (userId: string) => {
    try {
      await adminApi.verifyUser(userId);
      loadUsers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to verify user');
    }
  };

  const getKycBadgeClass = (status: string): string => {
    switch (status) {
      case 'approved': case 'verified': return 'badge-success';
      case 'pending': case 'level1': case 'level2': case 'level3': return 'badge-warning';
      case 'rejected': return 'badge-error';
      default: return 'badge-neutral';
    }
  };

  const getStatusBadgeClass = (status: string): string => {
    switch (status) {
      case 'active': return 'badge-success';
      case 'suspended': return 'badge-error';
      case 'banned': return 'badge-neutral';
      default: return 'badge-neutral';
    }
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
          User Management
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Manage platform users, KYC verification, and account status
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
                placeholder="Search by email or username..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
            <div className="w-40">
              <label className="form-label">Status</label>
              <select
                className="form-select"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="active">Active</option>
                <option value="suspended">Suspended</option>
                <option value="banned">Banned</option>
              </select>
            </div>
            <div className="w-40">
              <label className="form-label">KYC Status</label>
              <select
                className="form-select"
                value={kycFilter}
                onChange={(e) => setKycFilter(e.target.value)}
              >
                <option value="">All</option>
                <option value="none">Not Submitted</option>
                <option value="pending">Pending</option>
                <option value="approved">Approved</option>
                <option value="rejected">Rejected</option>
              </select>
            </div>
            <button type="submit" className="btn btn-primary">
              Search
            </button>
          </form>
        </div>
      </div>

      {/* Users Table */}
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
                    <th>Status</th>
                    <th>KYC</th>
                    <th>Volume</th>
                    <th>Joined</th>
                    <th>Last Login</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.length > 0 ? (
                    users.map((user) => (
                      <tr key={user.id}>
                        <td>
                          <div>
                            <p className="font-medium" style={{ color: 'var(--text-primary)' }}>
                              {user.username || user.email}
                            </p>
                            <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                              {user.email}
                            </p>
                          </div>
                        </td>
                        <td>
                          <span className={`badge ${getStatusBadgeClass(user.status)}`}>
                            {user.status}
                          </span>
                        </td>
                        <td>
                          <div className="flex items-center gap-2">
                            <span className={`badge ${getKycBadgeClass(user.kycStatus)}`}>
                              {user.kycStatus}
                            </span>
                            {user.verified && (
                              <span className="text-xs" style={{ color: 'var(--color-success)' }}>
                                ✓ Verified
                              </span>
                            )}
                          </div>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-primary)' }}>
                            ${parseFloat(user.totalVolume || '0').toLocaleString()}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {new Date(user.createdAt).toLocaleDateString()}
                          </span>
                        </td>
                        <td>
                          <span style={{ color: 'var(--text-secondary)' }}>
                            {user.lastLogin ? new Date(user.lastLogin).toLocaleDateString() : 'Never'}
                          </span>
                        </td>
                        <td>
                          <div className="flex gap-2">
                            <button
                              className="btn btn-sm btn-outline"
                              onClick={() => {
                                setSelectedUser(user);
                                setShowModal(true);
                              }}
                            >
                              Suspend
                            </button>
                            {user.suspended ? (
                              <button
                                className="btn btn-sm btn-success"
                                onClick={() => handleUnsuspendUser(user.id)}
                              >
                                Unsuspend
                              </button>
                            ) : (
                              !user.verified && (
                                <button
                                  className="btn btn-sm btn-secondary"
                                  onClick={() => handleVerifyUser(user.id)}
                                >
                                  Verify
                                </button>
                              )
                            )}
                          </div>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={7} className="text-center py-8" style={{ color: 'var(--text-tertiary)' }}>
                        No users found
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
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                const pageNum = i + 1;
                return (
                  <button
                    key={pageNum}
                    className={`pagination-btn ${page === pageNum ? 'active' : ''}`}
                    onClick={() => setPage(pageNum)}
                  >
                    {pageNum}
                  </button>
                );
              })}
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

      {/* Suspend Modal */}
      {showModal && selectedUser && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Suspend User</h3>
              <button
                className="btn btn-sm btn-outline"
                onClick={() => setShowModal(false)}
              >
                ✕
              </button>
            </div>
            <div className="modal-body">
              <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
                Are you sure you want to suspend user <strong>{selectedUser.email}</strong>?
              </p>
              <div className="form-group">
                <label className="form-label">Reason for suspension</label>
                <textarea
                  className="form-textarea"
                  rows={4}
                  value={suspendReason}
                  onChange={(e) => setSuspendReason(e.target.value)}
                  placeholder="Enter reason for suspension..."
                />
              </div>
            </div>
            <div className="modal-footer">
              <button
                className="btn btn-secondary"
                onClick={() => setShowModal(false)}
              >
                Cancel
              </button>
              <button
                className="btn btn-danger"
                onClick={handleSuspendUser}
                disabled={!suspendReason}
              >
                Suspend User
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default UsersPage;
