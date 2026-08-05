/**
 * KYC Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';

export default function KYC() {
  const [requests, setRequests] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadKYC(); }, []);

  const loadKYC = async () => {
    try {
      const data = await whiteLabelAdminApi.getKYCRequests();
      setRequests(data.data || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleApprove = async (id: string) => {
    try {
      await whiteLabelAdminApi.approveKYC(id);
      loadKYC();
    } catch (error) { console.error('Failed:', error); }
  };

  const handleReject = async (id: string) => {
    try {
      await whiteLabelAdminApi.rejectKYC(id, 'Rejected by admin');
      loadKYC();
    } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">KYC Management</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Document Type</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {requests.map((req) => (
              <tr key={req.id}>
                <td className="px-6 py-4">{req.userEmail}</td>
                <td className="px-6 py-4">{req.documentType}</td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${req.status === 'approved' ? 'bg-green-100 text-green-800' : req.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{req.status}</span></td>
                <td className="px-6 py-4 space-x-2">
                  {req.status === 'pending' && (
                    <>
                      <button onClick={() => handleApprove(req.id)} className="text-green-600 hover:underline">Approve</button>
                      <button onClick={() => handleReject(req.id)} className="text-red-600 hover:underline">Reject</button>
                    </>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
