import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { userService } from '../services/api';

interface User {
  id: string;
  user_id: string;
  username: string;
  email: string;
  phone: string | null;
  status: string;
  tier: number;
  kyc_status: string;
  kyc_level: number;
  is_email_verified: boolean;
  is_phone_verified: boolean;
  white_label_id: string | null;
  referral_code: string | null;
  created_at: string;
  last_login: string | null;
}

export default function Users() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    loadUsers();
  }, [page, statusFilter]);

  const loadUsers = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await userService.getUsers({
        page,
        limit: 20,
        status: statusFilter || undefined,
        search: search || undefined
      });
      
      setUsers(response.data);
      setTotalPages(response.meta.total_pages);
    } catch (err: any) {
      setError(err.message || 'Failed to load users');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    loadUsers();
  };

  const handleSuspend = async (userId: string) => {
    if (!confirm('Are you sure you want to suspend this user?')) return;
    
    setActionLoading(true);
    try {
      await userService.suspendUser(userId, 'Suspended by admin');
      alert('User suspended successfully');
      loadUsers();
    } catch (err: any) {
      alert(err.message || 'Failed to suspend user');
    } finally {
      setActionLoading(false);
    }
  };

  const handleBan = async (userId: string) => {
    if (!confirm('Are you sure you want to ban this user?')) return;
    
    setActionLoading(true);
    try {
      await userService.banUser(userId, 'Banned by admin');
      alert('User banned successfully');
      loadUsers();
    } catch (err: any) {
      alert(err.message || 'Failed to ban user');
    } finally {
      setActionLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800';
      case 'suspended': return 'bg-yellow-100 text-yellow-800';
      case 'banned': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getKYCColor = (status: string) => {
    switch (status) {
      case 'approved': return 'bg-green-100 text-green-800';
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      case 'rejected': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <div className="flex justify-between items-center mb-6">
          <h1 className={`text-3xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
            Users
          </h1>
          <button
            onClick={() => loadUsers()}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
          >
            Refresh
          </button>
        </div>

        {/* Filters */}
        <div className={`rounded-lg shadow p-4 mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          <form onSubmit={handleSearch} className="flex flex-wrap gap-4">
            <input
              type="text"
              placeholder="Search by username or email..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className={`flex-1 min-w-[200px] px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className={`px-4 py-2 rounded-lg border ${
                isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'
              }`}
            >
              <option value="">All Status</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="banned">Banned</option>
            </select>
            <button
              type="submit"
              className={`px-6 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
            >
              Search
            </button>
          </form>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {/* Users Table */}
        <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
          {loading ? (
            <div className="p-8 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-blue-500 border-t-transparent"></div>
              <p className={`mt-2 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Loading...</p>
            </div>
          ) : users.length === 0 ? (
            <div className="p-8 text-center">
              <p className={`${isDark ? 'text-gray-400' : 'text-gray-600'}`}>No users found</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
                  <tr>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>User</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>KYC</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Tier</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Verified</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Created</th>
                    <th className={`px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
                  </tr>
                </thead>
                <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
                  {users.map((user) => (
                    <tr key={user.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        <div>
                          <p className="font-medium">{user.username}</p>
                          <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{user.email}</p>
                        </div>
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(user.status)}`}>
                          {user.status}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${getKYCColor(user.kyc_status)}`}>
                          {user.kyc_status} (Lv.{user.kyc_level})
                        </span>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                        {user.tier}
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex gap-2">
                          {user.is_email_verified && <span className="text-green-500">✓ Email</span>}
                          {user.is_phone_verified && <span className="text-green-500">✓ Phone</span>}
                        </div>
                      </td>
                      <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                        {new Date(user.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex gap-2">
                          <button
                            onClick={() => setSelectedUser(user)}
                            className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
                          >
                            View
                          </button>
                          {user.status === 'active' && (
                            <>
                              <button
                                onClick={() => handleSuspend(user.id)}
                                disabled={actionLoading}
                                className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-yellow-600 hover:bg-yellow-700' : 'bg-yellow-500 hover:bg-yellow-600'} text-white`}
                              >
                                Suspend
                              </button>
                              <button
                                onClick={() => handleBan(user.id)}
                                disabled={actionLoading}
                                className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-red-600 hover:bg-red-700' : 'bg-red-500 hover:bg-red-600'} text-white`}
                              >
                                Ban
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

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

        {/* User Detail Modal */}
        {selectedUser && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className={`rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <h2 className={`text-2xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  User Details
                </h2>
                <button
                  onClick={() => setSelectedUser(null)}
                  className={`text-2xl ${isDark ? 'text-gray-400 hover:text-white' : 'text-gray-500 hover:text-gray-700'}`}
                >
                  ×
                </button>
              </div>
              
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Username</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.username}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Email</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.email}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Phone</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.phone || 'N/A'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Status</label>
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(selectedUser.status)}`}>
                      {selectedUser.status}
                    </span>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>KYC Status</label>
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getKYCColor(selectedUser.kyc_status)}`}>
                      {selectedUser.kyc_status}
                    </span>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Tier</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.tier}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Email Verified</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.is_email_verified ? 'Yes' : 'No'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Phone Verified</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.is_phone_verified ? 'Yes' : 'No'}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Created At</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{new Date(selectedUser.created_at).toLocaleString()}</p>
                  </div>
                  <div>
                    <label className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>Last Login</label>
                    <p className={`text-lg ${isDark ? 'text-white' : 'text-gray-900'}`}>{selectedUser.last_login ? new Date(selectedUser.last_login).toLocaleString() : 'N/A'}</p>
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
