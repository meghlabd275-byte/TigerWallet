/**
 * Withdrawals Page - White Label Admin
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';

export default function Withdrawals() {
  const [withdrawals, setWithdrawals] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadWithdrawals(); }, []);

  const loadWithdrawals = async () => {
    try {
      const data = await whiteLabelAdminApi.getWithdrawals();
      setWithdrawals(data.data || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  const handleApprove = async (id: string) => {
    try {
      await whiteLabelAdminApi.approveWithdrawal(id);
      loadWithdrawals();
    } catch (error) { console.error('Failed:', error); }
  };

  const handleReject = async (id: string) => {
    try {
      await whiteLabelAdminApi.rejectWithdrawal(id, 'Rejected by admin');
      loadWithdrawals();
    } catch (error) { console.error('Failed:', error); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Withdrawals</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Amount</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Address</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {withdrawals.map((w) => (
              <tr key={w.id}>
                <td className="px-6 py-4">{w.userEmail}</td>
                <td className="px-6 py-4">{w.amount} {w.token}</td>
                <td className="px-6 py-4 font-mono text-sm">{w.toAddress?.substring(0, 10)}...</td>
                <td className="px-6 py-4"><span className={`px-2 py-1 text-xs rounded ${w.status === 'approved' ? 'bg-green-100 text-green-800' : w.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{w.status}</span></td>
                <td className="px-6 py-4 space-x-2">
                  {w.status === 'pending' && (
                    <>
                      <button onClick={() => handleApprove(w.id)} className="text-green-600 hover:underline">Approve</button>
                      <button onClick={() => handleReject(w.id)} className="text-red-600 hover:underline">Reject</button>
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
