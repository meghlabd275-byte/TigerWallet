import React, { useEffect, useState } from 'react';
import { adminServicesApi } from '../services/api';

export default function Webhooks() {
  const [webhooks, setWebhooks] = useState<any[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ name: '', url: '', events: '', secret: '' });

  useEffect(() => { adminServicesApi.getWebhooks().then(r => setWebhooks(r.webhooks || [])).catch(console.error); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await adminServicesApi.createWebhook({ ...formData, events: formData.events.split(',') });
    setShowModal(false);
    setFormData({ name: '', url: '', events: '', secret: '' });
    adminServicesApi.getWebhooks().then(r => setWebhooks(r.webhooks || [])).catch(console.error);
  };

  const handleTest = async (id: string) => { await adminServicesApi.testWebhook(id); alert('Test webhook sent'); };
  const handleDelete = async (id: string) => { if (confirm('Delete webhook?')) { await adminServicesApi.deleteWebhook(id); adminServicesApi.getWebhooks().then(r => setWebhooks(r.webhooks || [])).catch(console.error); } };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Webhooks</h1>
        <button onClick={() => setShowModal(true)} className="bg-blue-600 text-white px-4 py-2 rounded">Create Webhook</button>
      </div>
      <div className="space-y-4">
        {webhooks.map(w => (
          <div key={w.id} className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold">{w.name}</h3>
                <p className="text-sm text-gray-500 font-mono">{w.url}</p>
                <p className="text-sm mt-1">Events: {w.events}</p>
                <p className={`text-sm mt-1 ${w.is_active ? 'text-green-600' : 'text-gray-500'}`}>{w.is_active ? 'Active' : 'Inactive'}</p>
              </div>
              <div className="space-x-2">
                <button onClick={() => handleTest(w.id)} className="text-blue-600 text-sm">Test</button>
                <button onClick={() => handleDelete(w.id)} className="text-red-600 text-sm">Delete</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create Webhook</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input type="text" placeholder="Name" value={formData.name} onChange={e => setFormData({...formData, name: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="url" placeholder="URL" value={formData.url} onChange={e => setFormData({...formData, url: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="text" placeholder="Events (comma-separated)" value={formData.events} onChange={e => setFormData({...formData, events: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
                <input type="text" placeholder="Secret" value={formData.secret} onChange={e => setFormData({...formData, secret: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
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
