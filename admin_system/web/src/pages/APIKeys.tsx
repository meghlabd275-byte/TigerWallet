/**
 * API Keys - Admin System
 */
import React, { useEffect, useState } from 'react';
import { adminSystemApi } from '../services/api';

export default function APIKeys() {
  const [keys, setKeys] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ name: '', permissions: [] as string[], expires_in: 30 });
  const [newKey, setNewKey] = useState<any>(null);

  useEffect(() => { loadKeys(); }, []);

  const loadKeys = async () => {
    try {
      const data = await adminSystemApi.getAPIKeys();
      setKeys(data.api_keys || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const result = await adminSystemApi.createAPIKey(formData);
      setNewKey(result.api_key);
      setShowModal(false);
      setFormData({ name: '', permissions: [], expires_in: 30 });
      loadKeys();
    } catch (error) { console.error('Failed:', error); }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this API key?')) return;
    try { await adminSystemApi.deleteAPIKey(id); loadKeys(); } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">API Keys</h1>
        <button onClick={() => setShowModal(true)} className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">Create API Key</button>
      </div>
      <div className="space-y-4">
        {keys.map((k) => (
          <div key={k.id} className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold">{k.name}</h3>
                <p className="text-sm font-mono text-gray-600 dark:text-gray-400 mt-1">{k.key}</p>
                <p className="text-xs text-gray-500 mt-2">Expires: {new Date(k.expires_at).toLocaleDateString()}</p>
              </div>
              <button onClick={() => handleDelete(k.id)} className="text-red-600 hover:underline">Delete</button>
            </div>
          </div>
        ))}
      </div>
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create API Key</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input type="text" placeholder="Name" value={formData.name} onChange={(e) => setFormData({...formData, name: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="number" placeholder="Expires in (days)" value={formData.expires_in} onChange={(e) => setFormData({...formData, expires_in: parseInt(e.target.value)})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
              </div>
              <div className="mt-4 flex justify-end space-x-2">
                <button type="button" onClick={() => setShowModal(false)} className="px-4 py-2 border rounded">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded">Create</button>
              </div>
            </form>
          </div>
        </div>
      )}
      {newKey && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">API Key Created</h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Save this key and secret - they won't be shown again!</p>
            <div className="bg-gray-100 dark:bg-gray-700 p-3 rounded mb-2">
              <p className="text-xs text-gray-500">Key:</p>
              <p className="font-mono text-sm">{newKey.key}</p>
            </div>
            <div className="bg-gray-100 dark:bg-gray-700 p-3 rounded mb-4">
              <p className="text-xs text-gray-500">Secret:</p>
              <p className="font-mono text-sm">{newKey.secret}</p>
            </div>
            <button onClick={() => setNewKey(null)} className="w-full px-4 py-2 bg-blue-600 text-white rounded">I have saved this</button>
          </div>
        </div>
      )}
    </div>
  );
}
