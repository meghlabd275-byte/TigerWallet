/**
 * Analytics Page
 * View analytics across all white labels
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';

export default function Analytics() {
  const [analytics, setAnalytics] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [period, setPeriod] = useState('24h');

  useEffect(() => {
    loadAnalytics();
  }, [period]);

  const loadAnalytics = async () => {
    try {
      const data = await masterAdminApi.getAnalytics(period);
      setAnalytics(data);
    } catch (error) {
      console.error('Failed to load analytics:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Analytics</h1>
        <select
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          className="px-3 py-2 border rounded dark:bg-gray-700"
        >
          <option value="24h">Last 24 Hours</option>
          <option value="7d">Last 7 Days</option>
          <option value="30d">Last 30 Days</option>
          <option value="90d">Last 90 Days</option>
        </select>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">Total Volume</h3>
          <p className="text-2xl font-bold mt-2">{analytics?.totalVolume || 0}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">Total Transactions</h3>
          <p className="text-2xl font-bold mt-2">{analytics?.totalTransactions || 0}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h3 className="text-sm text-gray-500 dark:text-gray-400">Active Users</h3>
          <p className="text-2xl font-bold mt-2">{analytics?.activeUsers || 0}</p>
        </div>
      </div>
    </div>
  );
}
