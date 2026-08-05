/**
 * Master Admins Management Page
 * Manage master admins across white labels
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';

export default function MasterAdmins() {
  const [admins, setAdmins] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ email: '', username: '', password: '', whiteLabelId: '' });

  useEffect(() => {
    loadAdmins();
  }, []);

  const loadAdmins = async () => {
    try {
      const data = await masterAdminApi.getMasterAdmins();
      setAdmins(data.data || []);
    } catch (error) {
      console.error('Failed to load master admins:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await masterAdminApi.createMasterAdmin(formData);
      setShowModal(false);
      setFormData({ email: '', username: '', password: '', whiteLabelId: '' });
      loadAdmins();
    } catch (error) {
      console.error('Failed to create master admin:', error);
    }
  };

  const handleSuspend = async (id: string) => {
    try {
      await masterAdminApi.suspendMasterAdmin(id);
      loadAdmins();
    } catch (error) {
      console.error('Failed to suspend master admin:', error);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Master Admins Management</h1>
        <button
          onClick={() => setShowModal(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
        >
          Create Master Admin
        </button>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Username</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">White Label</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {admins.map((admin) => (
              <tr key={admin.id}>
                <td className="px-6 py-4 whitespace-nowrap">{admin.username}</td>
                <td className="px-6 py-4">{admin.email}</td>
                <td className="px-6 py-4">{admin.whiteLabelId || 'All'}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs rounded ${
                    admin.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  }`}>
                    {admin.status}
                  </span>
                </td>
                <td className="px-6 py-4 space-x-2">
                  <button onClick={() => handleSuspend(admin.id)} className="text-red-600 hover:underline">Suspend</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create Master Admin</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input
                  type="text"
                  placeholder="Username"
                  value={formData.username}
                  onChange={(e) => setFormData({...formData, username: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                  required
                />
                <input
                  type="email"
                  placeholder="Email"
                  value={formData.email}
                  onChange={(e) => setFormData({...formData, email: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                  required
                />
                <input
                  type="password"
                  placeholder="Password"
                  value={formData.password}
                  onChange={(e) => setFormData({...formData, password: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                  required
                />
                <input
                  type="text"
                  placeholder="White Label ID (optional)"
                  value={formData.whiteLabelId}
                  onChange={(e) => setFormData({...formData, whiteLabelId: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div className="mt-4 flex justify-end space-x-2">
                <button type="button" onClick={() => setShowModal(false)} className="px-4 py-2 border rounded">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Create</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
