/**
 * Dashboard - Admin System
 */
import React, { useEffect, useState } from 'react';
import { adminSystemApi } from '../services/api';

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => { loadDashboard(); }, []);

  const loadDashboard = async () => {
    try {
      const data = await adminSystemApi.getDashboardStats();
      setStats(data.stats);
    } catch (error) { console.error('Failed:', error); }
    finally { setLoading(false); }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Admin System Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Total Users</h3><p className="text-2xl font-bold mt-2">{stats?.users_today || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Active Sessions</h3><p className="text-2xl font-bold mt-2">{stats?.active_sessions || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">API Calls Today</h3><p className="text-2xl font-bold mt-2">{stats?.api_calls_today || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Error Rate</h3><p className="text-2xl font-bold mt-2">{stats?.error_rate || 0}%</p></div>
      </div>
    </div>
  );
}
