/**
 * Audit Logs - Admin System
 */
import React, { useEffect, useState } from 'react';
import { adminSystemApi } from '../services/api';

export default function AuditLogs() {
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadLogs(); }, []);

  const loadLogs = async () => {
    try {
      const data = await adminSystemApi.getAuditLogs();
      setLogs(data.logs || []);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Audit Logs</h1>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">User</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Action</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Resource</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Details</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">IP Address</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Timestamp</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {logs.map((l) => (
              <tr key={l.id}>
                <td className="px-6 py-4">{l.user_email}</td>
                <td className="px-6 py-4"><span className="px-2 py-1 text-xs rounded bg-blue-100 text-blue-800">{l.action}</span></td>
                <td className="px-6 py-4">{l.resource}</td>
                <td className="px-6 py-4 text-sm">{l.details}</td>
                <td className="px-6 py-4 text-sm font-mono">{l.ip_address}</td>
                <td className="px-6 py-4 text-sm">{new Date(l.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
