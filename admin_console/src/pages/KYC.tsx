/**
 * KYC - Admin Console
 */
import React, { useEffect, useState } from 'react';
import { adminConsoleApi } from '../services/api';

export default function KYC() {
  const [requests, setRequests] = useState<any[]>([]);
  useEffect(() => { adminConsoleApi.getKYC().then(d => setRequests(d.data || [])).catch(console.error); }, []);
  const handleApprove = async (id: string) => { await adminConsoleApi.approveKYC(id); window.location.reload(); };
  const handleReject = async (id: string) => { await adminConsoleApi.rejectKYC(id, 'Rejected'); window.location.reload(); };
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">KYC Management</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700"><tr><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Type</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th><th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th></tr></thead>
          <tbody className="divide-y divide-gray-200">
            {requests.map(r => (<tr key={r.id}><td className="px-6 py-4">{r.userEmail}</td><td className="px-6 py-4">{r.documentType}</td><td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${r.status === 'approved' ? 'bg-green-100 text-green-800' : r.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{r.status}</span></td><td className="px-6 py-4 space-x-2">{r.status === 'pending' && (<><button onClick={() => handleApprove(r.id)} className="text-green-600 hover:underline">Approve</button><button onClick={() => handleReject(r.id)} className="text-red-600 hover:underline">Reject</button></>)}</td></tr>))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
