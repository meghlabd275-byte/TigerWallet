/**
 * Users Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';

export default function Users() {
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadUsers(); }, []);

  const loadUsers = async () => {
    try {
      const data = await whiteLabelAdminApi.getUsers({ pageSize: 50 });
      setUsers(data.data || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleSuspend = async (id: string) => {
    try {
      await whiteLabelAdminApi.suspendUser(id);
      loadUsers();
    } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Users Management</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">KYC Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {users.map((user) => (
              <tr key={user.id}>
                <td className="px-6 py-4">{user.id}</td>
                <td className="px-6 py-4">{user.email}</td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${user.kycStatus === 'verified' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>{user.kycStatus}</span></td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${user.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>{user.status}</span></td>
                <td className="px-6 py-4"><button onClick={() => handleSuspend(user.id)} className="text-red-600 hover:underline">Suspend</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
