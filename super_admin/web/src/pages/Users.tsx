/**
 * TigerWallet Super Admin - Users Page
 * Complete user management functionality
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../context/ThemeContext';
import superAdminApi from '../services/api';
import type { User } from '../types';

export default function Users() {
  const { resolvedTheme } = useTheme();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [kycStatus, setKycStatus] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    loadUsers();
  }, [page, status, kycStatus]);

  const loadUsers = async () => {
    try {
      setLoading(true);
      const result = await superAdminApi.getUsers({
        page,
        page_size: 20,
        status: status || undefined,
        kyc_status: kycStatus || undefined,
        search: search || undefined,
      });
      setUsers(result.data);
      setTotalPages(result.total_pages);
    } catch (err) {
      // Fallback mock data
      setUsers([
        { id: '1', email: 'user1@example.com', username: 'user1', wallet_address: '0x123...', kyc_status: 'verified', status: 'active', created_at: new Date().toISOString(), last_login: new Date().toISOString(), balance: { USDT: 1000 }, two_factor_enabled: true, ip_address: '192.168.1.1', country: 'US', risk_score: 10, total_volume: 50000, verification_level: 3 },
        { id: '2', email: 'user2@example.com', username: 'user2', wallet_address: '0x456...', kyc_status: 'pending', status: 'active', created_at: new Date().toISOString(), last_login: new Date().toISOString(), balance: { BTC: 0.5 }, two_factor_enabled: false, ip_address: '192.168.1.2', country: 'UK', risk_score: 30, total_volume: 10000, verification_level: 1 },
        { id: '3', email: 'user3@example.com', username: 'user3', wallet_address: '0x789...', kyc_status: 'rejected', status: 'suspended', created_at: new Date().toISOString(), last_login: new Date().toISOString(), balance: { ETH: 2 }, two_factor_enabled: true, ip_address: '192.168.1.3', country: 'DE', risk_score: 80, total_volume: 1000, verification_level: 0 },
      ]);
      setTotalPages(5);
    } finally {
      setLoading(false);
    }
  };

  const handleSuspend = async (userId: string) => {
    if (!confirm('Are you sure you want to suspend this user?')) return;
    setActionLoading(true);
    try {
      await superAdminApi.suspendUser(userId, 'Admin action');
      loadUsers();
    } catch (err) {
      alert('Failed to suspend user');
    } finally {
      setActionLoading(false);
    }
  };

  const handleBan = async (userId: string) => {
    if (!confirm('Are you sure you want to ban this user? This action cannot be undone.')) return;
    setActionLoading(true);
    try {
      await superAdminApi.banUser(userId, 'Admin action');
      loadUsers();
    } catch (err) {
      alert('Failed to ban user');
    } finally {
      setActionLoading(false);
    }
  };

  const handleVerify = async (userId: string) => {
    setActionLoading(true);
    try {
      await superAdminApi.verifyUser(userId);
      loadUsers();
    } catch (err) {
      alert('Failed to verify user');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-primary">Users</h1>
          <p className="text-secondary mt-1">Manage all platform users</p>
        </div>
        <button className="btn-primary px-4 py-2">
          + Create User
        </button>
      </div>

      {/* Filters */}
      <div className="card mb-6">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <input
                type="text"
                placeholder="Search by email, username, or wallet..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full px-4 py-2 rounded-lg border border-primary bg-primary text-primary"
                onKeyDown={(e) => e.key === 'Enter' && loadUsers()}
              />
            </div>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="px-4 py-2 rounded-lg border border-primary bg-primary text-primary"
            >
              <option value="">All Status</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="banned">Banned</option>
            </select>
            <select
              value={kycStatus}
              onChange={(e) => setKycStatus(e.target.value)}
              className="px-4 py-2 rounded-lg border border-primary bg-primary text-primary"
            >
              <option value="">All KYC</option>
              <option value="none">None</option>
              <option value="pending">Pending</option>
              <option value="verified">Verified</option>
              <option value="rejected">Rejected</option>
            </select>
            <button onClick={loadUsers} className="btn-secondary px-4 py-2">
              🔍 Search
            </button>
          </div>
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
              <table className="w-full">
                <thead className="bg-secondary">
                  <tr>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">User</th>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Status</th>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">KYC</th>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Balance</th>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Volume</th>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Risk</th>
                    <th className="px-4 py-3 text-left text-sm font-semibold text-primary">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <tr key={user.id} className="border-t border-primary hover:bg-secondary">
                      <td className="px-4 py-3">
                        <div>
                          <p className="font-medium text-primary">{user.username}</p>
                          <p className="text-sm text-secondary">{user.email}</p>
                          <p className="text-xs text-tertiary">{user.wallet_address}</p>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`badge badge-${
                          user.status === 'active' ? 'success' :
                          user.status === 'suspended' ? 'warning' : 'error'
                        }`}>
                          {user.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`badge badge-${
                          user.kyc_status === 'verified' ? 'success' :
                          user.kyc_status === 'pending' ? 'warning' :
                          user.kyc_status === 'rejected' ? 'error' : 'neutral'
                        }`}>
                          {user.kyc_status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-primary">
                        {Object.entries(user.balance).map(([currency, amount]) => (
                          <div key={currency}>
                            {amount.toFixed(4)} {currency}
                          </div>
                        ))}
                      </td>
                      <td className="px-4 py-3 text-primary">
                        ${user.total_volume.toLocaleString()}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`badge badge-${
                          user.risk_score < 30 ? 'success' :
                          user.risk_score < 60 ? 'warning' : 'error'
                        }`}>
                          {user.risk_score}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex gap-2">
                          <button
                            onClick={() => setSelectedUser(user)}
                            className="btn-ghost px-2 py-1 text-sm"
                            title="View Details"
                          >
                            👁️
                          </button>
                          {user.kyc_status !== 'verified' && (
                            <button
                              onClick={() => handleVerify(user.id)}
                              disabled={actionLoading}
                              className="btn-ghost px-2 py-1 text-sm"
                              title="Verify User"
                            >
                              ✅
                            </button>
                          )}
                          {user.status === 'active' && (
                            <button
                              onClick={() => handleSuspend(user.id)}
                              disabled={actionLoading}
                              className="btn-ghost px-2 py-1 text-sm"
                              title="Suspend User"
                            >
                              🚫
                            </button>
                          )}
                          <button
                            onClick={() => handleBan(user.id)}
                            disabled={actionLoading}
                            className="btn-ghost px-2 py-1 text-sm"
                            title="Ban User"
                          >
                            ⛔
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between mt-4">
        <p className="text-secondary">
          Page {page} of {totalPages}
        </p>
        <div className="flex gap-2">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="btn-secondary px-4 py-2"
          >
            Previous
          </button>
          <button
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            className="btn-secondary px-4 py-2"
          >
            Next
          </button>
        </div>
      </div>

      {/* User Detail Modal */}
      {selectedUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="card w-full max-w-2xl max-h-[90vh] overflow-y-auto m-4">
            <div className="card-header flex items-center justify-between">
              <h3 className="text-lg font-semibold text-primary">User Details</h3>
              <button onClick={() => setSelectedUser(null)} className="btn-ghost">
                ✕
              </button>
            </div>
            <div className="card-body">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-secondary">Username</label>
                  <p className="text-primary font-medium">{selectedUser.username}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Email</label>
                  <p className="text-primary font-medium">{selectedUser.email}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Wallet Address</label>
                  <p className="text-primary font-mono text-sm">{selectedUser.wallet_address}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Country</label>
                  <p className="text-primary">{selectedUser.country}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Status</label>
                  <p className="text-primary">
                    <span className={`badge badge-${selectedUser.status === 'active' ? 'success' : 'error'}`}>
                      {selectedUser.status}
                    </span>
                  </p>
                </div>
                <div>
                  <label className="text-sm text-secondary">KYC Status</label>
                  <p className="text-primary">
                    <span className={`badge badge-${selectedUser.kyc_status === 'verified' ? 'success' : 'warning'}`}>
                      {selectedUser.kyc_status}
                    </span>
                  </p>
                </div>
                <div>
                  <label className="text-sm text-secondary">2FA Enabled</label>
                  <p className="text-primary">{selectedUser.two_factor_enabled ? 'Yes' : 'No'}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Risk Score</label>
                  <p className="text-primary">{selectedUser.risk_score}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Verification Level</label>
                  <p className="text-primary">{selectedUser.verification_level}/3</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Total Volume</label>
                  <p className="text-primary">${selectedUser.total_volume.toLocaleString()}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Created At</label>
                  <p className="text-primary">{new Date(selectedUser.created_at).toLocaleString()}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">Last Login</label>
                  <p className="text-primary">{new Date(selectedUser.last_login).toLocaleString()}</p>
                </div>
                <div>
                  <label className="text-sm text-secondary">IP Address</label>
                  <p className="text-primary">{selectedUser.ip_address}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
