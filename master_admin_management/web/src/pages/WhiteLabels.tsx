/**
 * White Labels Management Page
 * Manage all white labels in the system
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';

export default function WhiteLabels() {
  const [whiteLabels, setWhiteLabels] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ name: '', domain: '', ownerEmail: '' });

  useEffect(() => {
    loadWhiteLabels();
  }, []);

  const loadWhiteLabels = async () => {
    try {
      const data = await masterAdminApi.getWhiteLabels();
      setWhiteLabels(data.data || []);
    } catch (error) {
      console.error('Failed to load white labels:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await masterAdminApi.createWhiteLabel(formData);
      setShowModal(false);
      setFormData({ name: '', domain: '', ownerEmail: '' });
      loadWhiteLabels();
    } catch (error) {
      console.error('Failed to create white label:', error);
    }
  };

  const handleApprove = async (id: string) => {
    try {
      await masterAdminApi.approveWhiteLabel(id);
      loadWhiteLabels();
    } catch (error) {
      console.error('Failed to approve white label:', error);
    }
  };

  const handleSuspend = async (id: string) => {
    try {
      await masterAdminApi.suspendWhiteLabel(id);
      loadWhiteLabels();
    } catch (error) {
      console.error('Failed to suspend white label:', error);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">White Labels Management</h1>
        <button
          onClick={() => setShowModal(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
        >
          Create White Label
        </button>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Domain</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Owner</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {whiteLabels.map((wl) => (
              <tr key={wl.id}>
                <td className="px-6 py-4 whitespace-nowrap">{wl.name}</td>
                <td className="px-6 py-4">{wl.domain}</td>
                <td className="px-6 py-4">{wl.ownerEmail}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs rounded ${
                    wl.status === 'active' ? 'bg-green-100 text-green-800' :
                    wl.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-red-100 text-red-800'
                  }`}>
                    {wl.status}
                  </span>
                </td>
                <td className="px-6 py-4 space-x-2">
                  {wl.status === 'pending' && (
                    <button onClick={() => handleApprove(wl.id)} className="text-blue-600 hover:underline">Approve</button>
                  )}
                  {wl.status === 'active' && (
                    <button onClick={() => handleSuspend(wl.id)} className="text-red-600 hover:underline">Suspend</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create White Label</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input
                  type="text"
                  placeholder="Name"
                  value={formData.name}
                  onChange={(e) => setFormData({...formData, name: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                  required
                />
                <input
                  type="text"
                  placeholder="Domain"
                  value={formData.domain}
                  onChange={(e) => setFormData({...formData, domain: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                  required
                />
                <input
                  type="email"
                  placeholder="Owner Email"
                  value={formData.ownerEmail}
                  onChange={(e) => setFormData({...formData, ownerEmail: e.target.value})}
                  className="w-full px-3 py-2 border rounded dark:bg-gray-700 dark:border-gray-600"
                  required
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
