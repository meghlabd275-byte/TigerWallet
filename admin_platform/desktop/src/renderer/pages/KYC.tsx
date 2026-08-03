import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { kycService } from '../services/api';

export default function KYC() {
  const { isDark } = useTheme();
  const [submissions, setSubmissions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadSubmissions(); }, []);

  const loadSubmissions = async () => {
    setLoading(true);
    try {
      const response = await kycService.getSubmissions({ page: 1, limit: 50 });
      setSubmissions(response.data);
    } catch (err) { console.error('Failed:', err); }
    finally { setLoading(false); }
  };

  const handleApprove = async (id: string) => {
    if (!confirm('Approve KYC?')) return;
    try { await kycService.approveKYC(id); loadSubmissions(); } catch (err) { alert('Failed'); }
  };

  const handleReject = async (id: string) => {
    const reason = prompt('Reason:');
    if (!reason) return;
    try { await kycService.rejectKYC(id, reason); loadSubmissions(); } catch (err) { alert('Failed'); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>KYC Submissions</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : submissions.length === 0 ? <div className="p-8 text-center">No submissions</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>User</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Level</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Country</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {submissions.map((sub) => (
                <tr key={sub.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{sub.first_name} {sub.last_name}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>Level {sub.level}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{sub.country || 'N/A'}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${sub.status === 'approved' ? 'bg-green-100 text-green-800' : sub.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{sub.status}</span></td>
                  <td className="px-4 py-4">
                    {sub.status === 'pending' && (
                      <div className="flex gap-2">
                        <button onClick={() => handleApprove(sub.id)} className="px-3 py-1 text-sm rounded bg-green-500 text-white">Approve</button>
                        <button onClick={() => handleReject(sub.id)} className="px-3 py-1 text-sm rounded bg-red-500 text-white">Reject</button>
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
