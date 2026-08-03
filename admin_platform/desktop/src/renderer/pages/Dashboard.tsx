import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { dashboardService } from '../services/api';

interface Stats {
  total_users: number;
  active_users: number;
  suspended_users: number;
  kyc_pending: number;
  total_transactions: number;
  volume_24h: number;
  revenue_24h: number;
  new_users_24h: number;
  new_transactions_24h: number;
}

export default function Dashboard() {
  const { isDark } = useTheme();
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    setLoading(true);
    try {
      const response = await dashboardService.getDashboard();
      setStats(response.data);
    } catch (err) {
      console.error('Failed to load dashboard:', err);
    } finally {
      setLoading(false);
    }
  };

  const statCards = [
    { title: 'Total Users', value: stats?.total_users ?? 0, icon: '👥', color: 'blue' },
    { title: 'Active Users', value: stats?.active_users ?? 0, icon: '✅', color: 'green' },
    { title: 'KYC Pending', value: stats?.kyc_pending ?? 0, icon: '⏳', color: 'yellow' },
    { title: 'Total Transactions', value: stats?.total_transactions ?? 0, icon: '💳', color: 'purple' },
    { title: '24h Volume', value: `$${(stats?.volume_24h ?? 0).toLocaleString()}`, icon: '📈', color: 'cyan' },
    { title: '24h Revenue', value: `$${(stats?.revenue_24h ?? 0).toLocaleString()}`, icon: '💰', color: 'gold' },
    { title: 'New Users (24h)', value: stats?.new_users_24h ?? 0, icon: '🆕', color: 'teal' },
    { title: 'New Transactions (24h)', value: stats?.new_transactions_24h ?? 0, icon: '🔄', color: 'orange' },
  ];

  const getColorClasses = (color: string) => {
    const colors: any = {
      blue: isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800',
      green: isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800',
      yellow: isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800',
      purple: isDark ? 'bg-purple-900 text-purple-200' : 'bg-purple-100 text-purple-800',
      cyan: isDark ? 'bg-cyan-900 text-cyan-200' : 'bg-cyan-100 text-cyan-800',
      gold: isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800',
      teal: isDark ? 'bg-teal-900 text-teal-200' : 'bg-teal-100 text-teal-800',
      orange: isDark ? 'bg-orange-900 text-orange-200' : 'bg-orange-100 text-orange-800',
    };
    return colors[color] || colors.blue;
  };

  if (loading) {
    return (
      <div className={`flex items-center justify-center h-64 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
        Loading...
      </div>
    );
  }

  return (
    <div>
      <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>
        Dashboard
      </h1>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {statCards.map((card, index) => (
          <div
            key={index}
            className={`p-6 rounded-lg shadow ${isDark ? 'bg-gray-800' : 'bg-white'}`}
          >
            <div className="flex items-center justify-between">
              <div>
                <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                  {card.title}
                </p>
                <p className={`text-2xl font-bold mt-1 ${isDark ? 'text-white' : 'text-gray-900'}`}>
                  {card.value}
                </p>
              </div>
              <div className={`p-3 rounded-lg ${getColorClasses(card.color)}`}>
                <span className="text-2xl">{card.icon}</span>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Quick Actions */}
      <div className={`p-6 rounded-lg shadow ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
        <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
          Quick Actions
        </h2>
        <div className="flex flex-wrap gap-4">
          <button
            onClick={() => window.location.href = '/users'}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-blue-600 hover:bg-blue-700' : 'bg-blue-500 hover:bg-blue-600'} text-white`}
          >
            Manage Users
          </button>
          <button
            onClick={() => window.location.href = '/kyc'}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-yellow-600 hover:bg-yellow-700' : 'bg-yellow-500 hover:bg-yellow-600'} text-white`}
          >
            Review KYC ({stats?.kyc_pending ?? 0})
          </button>
          <button
            onClick={() => window.location.href = '/withdrawals'}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-purple-600 hover:bg-purple-700' : 'bg-purple-500 hover:bg-purple-600'} text-white`}
          >
            Process Withdrawals
          </button>
          <button
            onClick={() => window.location.href = '/chains'}
            className={`px-4 py-2 rounded-lg ${isDark ? 'bg-green-600 hover:bg-green-700' : 'bg-green-500 hover:bg-green-600'} text-white`}
          >
            Manage Chains
          </button>
        </div>
      </div>
    </div>
  );
}
