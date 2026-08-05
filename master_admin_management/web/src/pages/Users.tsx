/**
 * Users Management Page
 * View and manage users across white labels
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';

export default function Users() {
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    try {
      const data = await masterAdminApi.getUsers({ pageSize: 50 });
      setUsers(data.data || []);
    } catch (error) {
      console.error('Failed to load users:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSuspend = async (id: string) => {
    try {
      await masterAdminApi.suspendUser(id);
      loadUsers();
    } catch (error) {
      console.error('Failed to suspend user:', error);
    }
  };

  const handleBan = async (id: string) => {
    try {
      await masterAdminApi.banUser(id);
      loadUsers();
    } catch (error) {
      console.error('Failed to ban user:', error);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Users Management</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">White Label</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {users.map((user) => (
              <tr key={user.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm">{user.id}</td>
                <td className="px-6 py-4">{user.email}</td>
                <td className="px-6 py-4">{user.whiteLabelId || 'N/A'}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs rounded ${
                    user.status === 'active' ? 'bg-green-100 text-green-800' :
                    user.status === 'suspended' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-red-100 text-red-800'
                  }`}>
                    {user.status}
                  </span>
                </td>
                <td className="px-6 py-4 space-x-2">
                  <button onClick={() => handleSuspend(user.id)} className="text-yellow-600 hover:underline">Suspend</button>
                  <button onClick={() => handleBan(user.id)} className="text-red-600 hover:underline">Ban</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
