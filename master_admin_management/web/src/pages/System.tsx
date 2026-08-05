/**
 * System Status Page
 * Monitor system health and metrics
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';

export default function System() {
  const [status, setStatus] = useState<any[]>([]);
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadSystemInfo();
  }, []);

  const loadSystemInfo = async () => {
    try {
      const [statusData, metricsData] = await Promise.all([
        masterAdminApi.getSystemStatus(),
        masterAdminApi.getSystemMetrics()
      ]);
      setStatus(statusData);
      setMetrics(metricsData);
    } catch (error) {
      console.error('Failed to load system info:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">System Status</h1>
      
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">CPU Usage</h3>
          <p className="text-2xl font-bold mt-2">{metrics?.cpuUsage || 0}%</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">Memory Usage</h3>
          <p className="text-2xl font-bold mt-2">{metrics?.memoryUsage || 0}%</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">Disk Usage</h3>
          <p className="text-2xl font-bold mt-2">{metrics?.diskUsage || 0}%</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">Uptime</h3>
          <p className="text-2xl font-bold mt-2">{metrics?.uptime || 'N/A'}</p>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Service</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">Latency</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {status.map((service) => (
              <tr key={service.name}>
                <td className="px-6 py-4 whitespace-nowrap">{service.name}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs rounded ${
                    service.status === 'running' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                  }`}>
                    {service.status}
                  </span>
                </td>
                <td className="px-6 py-4">{service.latency || 'N/A'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
