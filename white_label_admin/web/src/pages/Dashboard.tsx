/**
 * White Label Admin Dashboard
 */

import React, { useEffect, useState } from 'react';
import { whiteLabelAdminApi } from '../services/api';
import { useTheme } from '../context/ThemeContext';

export default function Dashboard() {
  const { isDark } = useTheme();
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => { loadDashboard(); }, []);

  const loadDashboard = async () => {
    try {
      const data = await whiteLabelAdminApi.getDashboardStats();
      setStats(data);
    } catch (e: any) { setError(e.message || 'Failed to load stats'); }
    finally { setLoading(false); }
  };

  const cardBg = isDark ? 'bg-gray-800' : 'bg-white';
  const cardText = isDark ? 'text-white' : 'text-gray-900';
  const muted = isDark ? 'text-gray-400' : 'text-gray-500';

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className={`p-8 ${muted}`}>Error: {error}</div>;

  const cards = [
    { label: 'Total Users', value: stats?.total_users ?? 0 },
    { label: 'Active Users', value: stats?.active_users ?? 0 },
    { label: 'Total Tokens', value: stats?.total_tokens ?? 0 },
    { label: 'Trading Pairs', value: stats?.total_pairs ?? 0 },
    { label: 'Open Tickets', value: stats?.open_tickets ?? 0 },
    { label: 'Pending KYC', value: stats?.pending_kyc ?? 0 },
  ];

  return (
    <div className="p-6">
      <h1 className={`text-2xl font-bold mb-6 ${cardText}`}>White Label Admin Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {cards.map((c) => (
          <div key={c.label} className={`${cardBg} p-6 rounded-lg shadow`}>
            <h3 className={`text-sm ${muted}`}>{c.label}</h3>
            <p className={`text-2xl font-bold mt-2 ${cardText}`}>{c.value}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
