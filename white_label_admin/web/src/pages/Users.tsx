/**
 * Users Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Users() {
  const { isDark } = useTheme();
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { loadUsers(); }, []);

  const loadUsers = async () => {
    setLoading(true); setError('');
    try {
      const data = await whiteLabelAdminApi.getUsers();
      setUsers(data.users || []);
    } catch (e: any) { setError(e.message || 'Failed to load users'); }
    finally { setLoading(false); }
  };

  const handleSuspend = async (id: string) => {
    try { await whiteLabelAdminApi.suspendUser(id); loadUsers(); }
    catch (e: any) { setError(e.message || 'Failed to suspend user'); }
  };

  const handleBan = async (id: string) => {
    try { await whiteLabelAdminApi.banUser(id); loadUsers(); }
    catch (e: any) { setError(e.message || 'Failed to ban user'); }
  };

  const handleUnban = async (id: string) => {
    try { await whiteLabelAdminApi.unbanUser(id); loadUsers(); }
    catch (e: any) { setError(e.message || 'Failed to unban user'); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const border = isDark ? 'border-gray-700' : 'border-gray-200';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';
  const thBg = isDark ? 'bg-gray-700' : 'bg-gray-50';

  const badge = (status: string) => {
    const base = 'px-2 py-1 text-xs rounded';
    if (status === 'active') return `${base} ${isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800'}`;
    if (status === 'banned') return `${base} ${isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800'}`;
    return `${base} ${isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800'}`;
  };

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-6 ${cardText}`}>Users Management</h1>
      {error && <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/50 text-red-200' : 'bg-red-50 text-red-700'}`}>{error}</div>}
      {loading && <div className={muted}>Loading...</div>}
      {!loading && (
        <div className={`${cardBg} rounded-lg shadow overflow-hidden border ${border}`}>
          <table className="w-full">
            <thead className={thBg}>
              <tr>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Email</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Username</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>KYC</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Status</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Country</th>
                <th className={`px-6 py-3 text-left text-xs font-medium ${muted} uppercase`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${border}`}>
              {users.length === 0 && (
                <tr><td colSpan={6} className={`px-6 py-8 text-center ${muted}`}>No users found.</td></tr>
              )}
              {users.map((user) => (
                <tr key={user.id}>
                  <td className={`px-6 py-4 ${cardText}`}>{user.email}</td>
                  <td className={`px-6 py-4 ${cardText}`}>{user.username}</td>
                  <td className="px-6 py-4"><span className={badge(user.kyc_status)}>{user.kyc_status || '—'}</span></td>
                  <td className="px-6 py-4"><span className={badge(user.status)}>{user.status}</span></td>
                  <td className={`px-6 py-4 ${muted}`}>{user.country || '—'}</td>
                  <td className="px-6 py-4 space-x-2">
                    {user.status !== 'banned' && <button onClick={() => handleBan(user.id)} className="text-red-600 hover:underline text-sm">Ban</button>}
                    {user.status === 'banned' && <button onClick={() => handleUnban(user.id)} className="text-green-600 hover:underline text-sm">Unban</button>}
                    {user.status !== 'suspended' && <button onClick={() => handleSuspend(user.id)} className="text-yellow-600 hover:underline text-sm">Suspend</button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
