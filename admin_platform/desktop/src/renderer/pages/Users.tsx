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
  created_at: string;
  last_login: string | null;
}

export default function Users() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [page, setPage] = useState(1);

  useEffect(() => {
    loadUsers();
  }, [page, statusFilter]);

  const loadUsers = async () => {
    setLoading(true);
    try {
      const response = await userService.getUsers({ page, limit: 20, status: statusFilter || undefined, search: search || undefined });
      setUsers(response.data);
    } catch (err) {
      console.error('Failed to load users:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSuspend = async (userId: string) => {
    if (!confirm('Suspend this user?')) return;
    try {
      await userService.suspendUser(userId, 'Suspended by admin');
      loadUsers();
    } catch (err) {
      alert('Failed to suspend user');
    }
  };

  const handleBan = async (userId: string) => {
    if (!confirm('Ban this user?')) return;
    try {
      await userService.banUser(userId, 'Banned by admin');
      loadUsers();
    } catch (err) {
      alert('Failed to ban user');
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

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Users</h1>
      
      <div className={`p-4 rounded-lg shadow mb-6 ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        <div className="flex flex-wrap gap-4">
          <input
            type="text"
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && loadUsers()}
            className={`flex-1 px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
          />
          <select
            value={statusFilter}
            onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
            className={`px-4 py-2 rounded-lg border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="banned">Banned</option>
          </select>
          <button onClick={loadUsers} className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 text-white' : 'bg-blue-500 text-white'}`}>Search</button>
        </div>
      </div>

      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? (
          <div className="p-8 text-center">Loading...</div>
        ) : users.length === 0 ? (
          <div className="p-8 text-center">No users found</div>
        ) : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>User</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>KYC</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Tier</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
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
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(user.status)}`}>{user.status}</span>
                  </td>
                  <td className="px-4 py-4">
                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${user.kyc_status === 'approved' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                      {user.kyc_status} (Lv.{user.kyc_level})
                    </span>
                  </td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{user.tier}</td>
                  <td className="px-4 py-4">
                    {user.status === 'active' && (
                      <div className="flex gap-2">
                        <button onClick={() => handleSuspend(user.id)} className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-yellow-600' : 'bg-yellow-500'} text-white`}>Suspend</button>
                        <button onClick={() => handleBan(user.id)} className={`px-3 py-1 text-sm rounded ${isDark ? 'bg-red-600' : 'bg-red-500'} text-white`}>Ban</button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {users.length > 0 && (
        <div className="flex justify-center gap-2 mt-6">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 text-white' : 'bg-white text-gray-700'} disabled:opacity-50`}>Previous</button>
          <span className={`px-4 py-2 ${isDark ? 'text-white' : 'text-gray-700'}`}>Page {page}</span>
          <button onClick={() => setPage(p => p + 1)} className={`px-4 py-2 rounded-lg ${isDark ? 'bg-gray-700 text-white' : 'bg-white text-gray-700'}`}>Next</button>
        </div>
      )}
    </div>
  );
}
