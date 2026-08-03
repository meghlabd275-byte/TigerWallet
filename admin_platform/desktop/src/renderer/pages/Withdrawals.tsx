import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { withdrawalService } from '../services/api';

export default function Withdrawals() {
  const { isDark } = useTheme();
  const [withdrawals, setWithdrawals] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadWithdrawals(); }, []);

  const loadWithdrawals = async () => {
    setLoading(true);
    try {
      const response = await withdrawalService.getWithdrawals({ page: 1, limit: 50 });
      setWithdrawals(response.data);
    } catch (err) { console.error('Failed:', err); }
    finally { setLoading(false); }
  };

  const handleApprove = async (id: string) => {
    if (!confirm('Approve withdrawal?')) return;
    try { await withdrawalService.approveWithdrawal(id); loadWithdrawals(); } catch (err) { alert('Failed'); }
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Reason:');
    if (!reason) return;
    try { await withdrawalService.rejectWithdrawal(id, reason); loadWithdrawals(); } catch (err) { alert('Failed'); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Withdrawals</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : withdrawals.length === 0 ? <div className="p-8 text-center">No withdrawals</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>User</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Token</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Amount</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {withdrawals.map((w) => (
                <tr key={w.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{w.user_id.substring(0, 8)}...</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{w.token}</td>
                  <td className={`px-4 py-4 font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{w.amount}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${w.status === 'approved' ? 'bg-green-100 text-green-800' : w.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{w.status}</span></td>
                  <td className="px-4 py-4">
                    {w.status === 'pending' && (
                      <div className="flex gap-2">
                        <button onClick={() => handleApprove(w.id)} className="px-3 py-1 text-sm rounded bg-green-500 text-white">Approve</button>
                        <button onClick={() => handleReject(w.id)} className="px-3 py-1 text-sm rounded bg-red-500 text-white">Reject</button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
