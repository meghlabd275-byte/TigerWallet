/**
 * Dashboard - Admin Console
 */
import React, { useEffect, useState } from 'react';
import { adminConsoleApi } from '../services/api';

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null);
  useEffect(() => { adminConsoleApi.getDashboardStats().then(setStats).catch(console.error); }, []);
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Admin Console Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Total Users</h3><p className="text-2xl font-bold mt-2">{stats?.totalUsers || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Transactions</h3><p className="text-2xl font-bold mt-2">{stats?.totalTransactions || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Volume</h3><p className="text-2xl font-bold mt-2">{stats?.totalVolume || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Pending KYC</h3><p className="text-2xl font-bold mt-2">{stats?.pendingKYC || 0}</p></div>
      </div>
    </div>
  );
}
