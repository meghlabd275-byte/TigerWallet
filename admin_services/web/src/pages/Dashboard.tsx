import React, { useEffect, useState } from 'react';
import { adminServicesApi } from '../services/api';

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null);
  useEffect(() => { adminServicesApi.getDashboardStats().then(r => setStats(r.stats)).catch(console.error); }, []);
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Admin Services Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Total Services</h3><p className="text-2xl font-bold mt-2">{stats?.services_today || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Running Services</h3><p className="text-2xl font-bold mt-2">{stats?.running_services || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Webhooks Fired</h3><p className="text-2xl font-bold mt-2">{stats?.webhooks_fired || 0}</p></div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow"><h3 className="text-sm text-gray-500">Jobs Completed</h3><p className="text-2xl font-bold mt-2">{stats?.jobs_completed || 0}</p></div>
      </div>
    </div>
  );
}
