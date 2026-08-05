/**
 * Configuration - Admin System
 */
import React, { useEffect, useState } from 'react';
import { adminSystemApi } from '../services/api';

export default function Config() {
  const [configs, setConfigs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ key: '', value: '', description: '', category: 'general' });

  useEffect(() => { loadConfig(); }, []);

  const loadConfig = async () => {
    try {
      const data = await adminSystemApi.getConfig();
      setConfigs(data.configs || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await adminSystemApi.updateConfig(formData);
      setShowModal(false);
      setFormData({ key: '', value: '', description: '', category: 'general' });
      loadConfig();
    } catch (error) { console.error('Failed:', error); }
  };

  const handleDelete = async (key: string) => {
    if (!confirm('Delete this config?')) return;
    try { await adminSystemApi.deleteConfig(key); loadConfig(); } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">System Configuration</h1>
        <button onClick={() => setShowModal(true)} className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">Add Config</button>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Value</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Category</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {configs.map((c) => (
              <tr key={c.id}>
                <td className="px-6 py-4 font-mono text-sm">{c.key}</td>
                <td className="px-6 py-4">{c.value.substring(0, 50)}...</td>
                <td className="px-6 py-4">{c.category}</td>
                <td className="px-6 py-4"><button onClick={() => handleDelete(c.key)} className="text-red-600 hover:underline">Delete</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Add Configuration</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input type="text" placeholder="Key" value={formData.key} onChange={(e) => setFormData({...formData, key: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <textarea placeholder="Value" value={formData.value} onChange={(e) => setFormData({...formData, value: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="text" placeholder="Category" value={formData.category} onChange={(e) => setFormData({...formData, category: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
              </div>
              <div className="mt-4 flex justify-end space-x-2">
                <button type="button" onClick={() => setShowModal(false)} className="px-4 py-2 border rounded">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
