import React, { useEffect, useState } from 'react';
import { adminServicesApi } from '../services/api';

export default function Jobs() {
  const [jobs, setJobs] = useState<any[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({ name: '', type: '', schedule: '' });

  useEffect(() => { adminServicesApi.getJobs().then(r => setJobs(r.jobs || [])).catch(console.error); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await adminServicesApi.createJob(formData);
    setShowModal(false);
    setFormData({ name: '', type: '', schedule: '' });
    adminServicesApi.getJobs().then(r => setJobs(r.jobs || [])).catch(console.error);
  };

  const handleRun = async (id: string) => { await adminServicesApi.runJob(id); adminServicesApi.getJobs().then(r => setJobs(r.jobs || [])).catch(console.error); };
  const handleDelete = async (id: string) => { if (confirm('Delete job?')) { await adminServicesApi.deleteJob(id); adminServicesApi.getJobs().then(r => setJobs(r.jobs || [])).catch(console.error); } };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Scheduled Jobs</h1>
        <button onClick={() => setShowModal(true)} className="bg-blue-600 text-white px-4 py-2 rounded">Create Job</button>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Schedule</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {jobs.map(j => (
              <tr key={j.id}>
                <td className="px-6 py-4">{j.name}</td>
                <td className="px-6 py-4">{j.type}</td>
                <td className="px-6 py-4 font-mono text-sm">{j.schedule || 'Manual'}</td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${j.status === 'running' ? 'bg-green-100 text-green-800' : j.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-gray-100 text-gray-800'}`}>{j.status}</span></td>
                <td className="px-6 py-4 space-x-2">
                  <button onClick={() => handleRun(j.id)} className="text-blue-600 hover:underline">Run</button>
                  <button onClick={() => handleDelete(j.id)} className="text-red-600 hover:underline">Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create Job</h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <input type="text" placeholder="Name" value={formData.name} onChange={e => setFormData({...formData, name: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="text" placeholder="Type" value={formData.type} onChange={e => setFormData({...formData, type: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" required />
                <input type="text" placeholder="Schedule (cron)" value={formData.schedule} onChange={e => setFormData({...formData, schedule: e.target.value})} className="w-full px-3 py-2 border rounded dark:bg-gray-700" />
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
