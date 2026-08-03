import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { whiteLabelService } from '../services/api';

export default function WhiteLabels() {
  const { isDark } = useTheme();
  const [whiteLabels, setWhiteLabels] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadWhiteLabels(); }, []);

  const loadWhiteLabels = async () => {
    setLoading(true);
    try {
      const response = await whiteLabelService.getWhiteLabels({ page: 1, limit: 50 });
      setWhiteLabels(response.data);
    } catch (err) { console.error('Failed:', err); }
    finally { setLoading(false); }
  };

  const handleApprove = async (id: string) => {
    try { await whiteLabelService.approveWhiteLabel(id); loadWhiteLabels(); } catch (err) { alert('Failed'); }
  };

  const handleSuspend = async (id: string) => {
    const reason = prompt('Reason:');
    if (!reason) return;
    try { await whiteLabelService.suspendWhiteLabel(id, reason); loadWhiteLabels(); } catch (err) { alert('Failed'); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>White Labels</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : whiteLabels.length === 0 ? <div className="p-8 text-center">No white labels</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Client</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Domain</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Verified</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Users</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Actions</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {whiteLabels.map((wl) => (
                <tr key={wl.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}><div><p className="font-medium">{wl.name}</p><p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>ID: {wl.client_id}</p></div></td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{wl.domain}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${wl.domain_verified ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>{wl.domain_verified ? 'Yes' : 'No'}</span></td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${wl.status === 'active' ? 'bg-green-100 text-green-800' : wl.status === 'pending' ? 'bg-yellow-100 text-yellow-800' : 'bg-red-100 text-red-800'}`}>{wl.status}</span></td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{wl.current_users} / {wl.max_users}</td>
                  <td className="px-4 py-4">
                    <div className="flex gap-2">
                      {wl.status === 'pending' && <button onClick={() => handleApprove(wl.id)} className="px-3 py-1 text-sm rounded bg-green-500 text-white">Approve</button>}
                      {wl.status === 'active' && <button onClick={() => handleSuspend(wl.id)} className="px-3 py-1 text-sm rounded bg-red-500 text-white">Suspend</button>}
                    </div>
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
