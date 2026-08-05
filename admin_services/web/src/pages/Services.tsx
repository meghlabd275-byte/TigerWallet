import React, { useEffect, useState } from 'react';
import { adminServicesApi } from '../services/api';

export default function Services() {
  const [services, setServices] = useState<any[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ name: '', type: '', endpoint: '' });

  useEffect(() => { adminServicesApi.getServices().then(r => setServices(r.services || [])).catch(console.error); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await adminServicesApi.createService(formData);
    setShowModal(false);
    setFormData({ name: '', type: '', endpoint: '' });
    adminServicesApi.getServices().then(r => setServices(r.services || [])).catch(console.error);
  };

  const handleStart = async (id: string) => { await adminServicesApi.startService(id); adminServicesApi.getServices().then(r => setServices(r.services || [])).catch(console.error); };
  const handleStop = async (id: string) => { await adminServicesApi.stopService(id); adminServicesApi.getServices().then(r => setServices(r.services || [])).catch(console.error); };
  const handleDelete = async (id: string) => { if (confirm('Delete service?')) { await adminServicesApi.deleteService(id); adminServicesApi.getServices().then(r => setServices(r.services || [])).catch(console.error); } };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Services Management</h1>
        <button onClick={() => setShowModal(true)} className="bg-blue-600 text-white px-4 py-2 rounded">Create Service</button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {services.map(s => (
          <div key={s.id} className="bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
            <div className="flex justify-between items-start">
              <div>
                <h3 className="font-semibold">{s.name}</h3>
                <p className="text-sm text-gray-500">{s.type} - {s.endpoint}</p>
                <p className={`text-sm mt-2 ${s.status === 'running' ? 'text-green-600' : 'text-red-600'}`}>Status: {s.status}</p>
              </div>
              <div className="space-x-2">
                {s.status !== 'running' && <button onClick={() => handleStart(s.id)} className="text-green-600 text-sm">Start</button>}
                {s.status === 'running' && <button onClick={() => handleStop(s.id)} className="text-yellow-600 text-sm">Stop</button>}
                <button onClick={() => handleDelete(s.id)} className="text-red-600 text-sm">Delete</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create Service</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input type="text" placeholder="Name" value={formData.name} onChange={e => setFormData({...formData, name: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="text" placeholder="Type" value={formData.type} onChange={e => setFormData({...formData, type: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="text" placeholder="Endpoint" value={formData.endpoint} onChange={e => setFormData({...formData, endpoint: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
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
