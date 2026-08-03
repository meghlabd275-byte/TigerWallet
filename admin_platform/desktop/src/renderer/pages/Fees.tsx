import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { feeService } from '../services/api';

export default function Fees() {
  const { isDark } = useTheme();
  const [fees, setFees] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadFees(); }, []);

  const loadFees = async () => {
    setLoading(true);
    try {
      const response = await feeService.getFees();
      setFees(response.data);
    } catch (err) { console.error('Failed:', err); }
    finally { setLoading(false); }
  };

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>Fee Configuration</h1>
      <div className={`rounded-lg shadow overflow-hidden ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        {loading ? <div className="p-8 text-center">Loading...</div> : fees.length === 0 ? <div className="p-8 text-center">No fees</div> : (
          <table className="w-full">
            <thead className={isDark ? 'bg-gray-700' : 'bg-gray-50'}>
              <tr>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Type</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Chain</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Percent</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Fixed</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Min</th>
                <th className={`px-4 py-3 text-left text-xs font-medium ${isDark ? 'text-gray-300' : 'text-gray-600'}`}>Status</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
              {fees.map((fee) => (
                <tr key={fee.id} className={isDark ? 'hover:bg-gray-700' : 'hover:bg-gray-50'}>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${fee.fee_type === 'deposit' ? 'bg-green-100 text-green-800' : fee.fee_type === 'withdrawal' ? 'bg-red-100 text-red-800' : 'bg-blue-100 text-blue-800'}`}>{fee.fee_type}</span></td>
                  <td className={`px-4 py-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>{fee.chain_id || 'All'}</td>
                  <td className={`px-4 py-4 font-medium ${isDark ? 'text-white' : 'text-gray-900'}`}>{fee.fee_percent}%</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{fee.fee_fixed}</td>
                  <td className={`px-4 py-4 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{fee.min_fee}</td>
                  <td className="px-4 py-4"><span className={`px-2 py-1 text-xs font-medium rounded-full ${fee.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>{fee.is_active ? 'Active' : 'Inactive'}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
