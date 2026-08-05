/**
 * Users - Admin Console
 */
import React, { useEffect, useState } from 'react';
import { adminConsoleApi } from '../services/api';

export default function Users() {
  const [users, setUsers] = useState<any[]>([]);
  useEffect(() => { adminConsoleApi.getUsers().then(d => setUsers(d.data || [])).catch(console.error); }, []);
  const handleSuspend = async (id: string) => { await adminConsoleApi.suspendUser(id); window.location.reload(); };
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Users</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700"><tr><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th></tr></thead>
          <tbody className="divide-y divide-gray-200">
            {users.map(u => (<tr key={u.id}><td className="px-6 py-4">{u.id}</td><td className="px-6 py-4">{u.email}</td><td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${u.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>{u.status}</span></td><td className="px-6 py-4"><button onClick={() => handleSuspend(u.id)} className="text-red-600 hover:underline">Suspend</button></td></tr>))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
