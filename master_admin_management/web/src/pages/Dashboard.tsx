/**
 * Master Admin Dashboard Page
 * Main dashboard with overview statistics
 */

import React, { useEffect, useState } from 'react';
import { masterAdminApi } from '../services/api';

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadDashboard();
  }, []);

  const loadDashboard = async () => {
    try {
      const data = await masterAdminApi.getDashboardStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Master Admin Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatCard title="Total White Labels" value={stats?.totalWhiteLabels || 0} />
        <StatCard title="Total Users" value={stats?.totalUsers || 0} />
        <StatCard title="Total Transactions" value={stats?.totalTransactions || 0} />
        <StatCard title="Active Master Admins" value={stats?.totalMasterAdmins || 0} />
      </div>
      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
          <p className="text-gray-600 dark:text-gray-400">No recent activity</p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-4">System Status</h2>
          <p className="text-green-600">All systems operational</p>
        </div>
      </div>
    </div>
  );
}

function StatCard({ title, value }: { title: string; value: number | string }) {
  return (
    <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow">
      <h3 className="text-sm text-gray-500 dark:text-gray-400">{title}</h3>
      <p className="text-2xl font-bold mt-2">{value}</p>
    </div>
  );
}
