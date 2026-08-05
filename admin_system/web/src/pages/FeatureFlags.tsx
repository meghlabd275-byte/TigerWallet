/**
 * Feature Flags - Admin System
 */
import React, { useEffect, useState } from 'react';
import { adminSystemApi } from '../services/api';

export default function FeatureFlags() {
  const [features, setFeatures] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ name: '', description: '', is_enabled: false, rollout_percent: 0 });

  useEffect(() => { loadFeatures(); }, []);

  const loadFeatures = async () => {
    try {
      const data = await adminSystemApi.getFeatureFlags();
      setFeatures(data.features || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await adminSystemApi.createFeatureFlag(formData);
      setShowModal(false);
      setFormData({ name: '', description: '', is_enabled: false, rollout_percent: 0 });
      loadFeatures();
    } catch (error) { console.error('Failed:', error); }
  };

  const handleToggle = async (id: string, currentState: boolean) => {
    try { await adminSystemApi.updateFeatureFlag(id, { is_enabled: !currentState }); loadFeatures(); } catch (error) { console.error('Failed:', error); }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this feature flag?')) return;
    try { await adminSystemApi.deleteFeatureFlag(id); loadFeatures(); } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Feature Flags</h1>
        <button onClick={() => setShowModal(true)} className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">Create Flag</button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {features.map((f) => (
          <div key={f.id} className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold">{f.name}</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{f.description}</p>
                <p className="text-xs text-gray-500 mt-2">Rollout: {f.rollout_percent}%</p>
              </div>
              <div className="flex items-center space-x-2">
                <button onClick={() => handleToggle(f.id, f.is_enabled)} className={`px-3 py-1 rounded text-sm ${f.is_enabled ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                  {f.is_enabled ? 'Enabled' : 'Disabled'}
                </button>
                <button onClick={() => handleDelete(f.id)} className="text-red-600 hover:underline text-sm">Delete</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create Feature Flag</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input type="text" placeholder="Name" value={formData.name} onChange={(e) => setFormData({...formData, name: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <textarea placeholder="Description" value={formData.description} onChange={(e) => setFormData({...formData, description: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
                <div className="flex items-center space-x-4">
                  <label className="flex items-center space-x-2">
                    <input type="checkbox" checked={formData.is_enabled} onChange={(e) => setFormData({...formData, is_enabled: e.target.checked})} />
                    <span>Enabled</span>
                  </label>
                </div>
                <input type="number" placeholder="Rollout %" value={formData.rollout_percent} onChange={(e) => setFormData({...formData, rollout_percent: parseInt(e.target.value)})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
              </div>
              <div className="mt-4 flex justify-end space-x-2">
                <button type="button" onClick={() => setShowModal(false)} className="px-4 py-2 border rounded">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded">Create</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
