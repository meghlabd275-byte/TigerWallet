import React, { useEffect, useState } from 'react';
import { useTheme } from '../context/ThemeContext';
import { dashboardService } from '../services/api';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';

interface Stats {
  totalUsers: number;
  activeUsers: number;
  suspendedUsers: number;
  kycPending: number;
  totalTransactions: number;
  volume24h: number;
  revenue24h: number;
}

interface ChartData {
  name: string;
  users: number;
  transactions: number;
}

export default function Dashboard() {
  const { isDark } = useTheme();
  const [stats, setStats] = useState<Stats | null>(null);
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadDashboard();
  }, []);

  const loadDashboard = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await dashboardService.getStats();
      setStats({
        totalUsers: response.data.totalUsers,
        activeUsers: response.data.activeUsers,
        suspendedUsers: response.data.suspendedUsers,
        kycPending: response.data.kycPending,
        totalTransactions: response.data.totalTransactions,
        volume24h: response.data.volume24h,
        revenue24h: response.data.revenue24h,
      });

      // Mock chart data - in production, fetch from analytics API
      const mockData = [
        { name: 'Mon', users: 120, transactions: 450 },
        { name: 'Tue', users: 145, transactions: 520 },
        { name: 'Wed', users: 132, transactions: 480 },
        { name: 'Thu', users: 168, transactions: 610 },
        { name: 'Fri', users: 195, transactions: 720 },
        { name: 'Sat', users: 210, transactions: 680 },
        { name: 'Sun', users: 188, transactions: 590 },
      ];
      setChartData(mockData);
    } catch (err: any) {
      setError(err.message || 'Failed to load dashboard');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className={`flex items-center justify-center h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        <div className="text-blue-600 text-xl">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`flex items-center justify-center h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
        <div className="text-red-600 text-xl">{error}</div>
      </div>
    );
  }

  const statCards = [
    { title: 'Total Users', value: stats?.totalUsers || 0, color: 'blue' },
    { title: 'Active Users', value: stats?.activeUsers || 0, color: 'green' },
    { title: 'Pending KYC', value: stats?.kycPending || 0, color: 'yellow' },
    { title: 'Total Transactions', value: stats?.totalTransactions || 0, color: 'purple' },
    { title: '24h Volume', value: `$${(stats?.volume24h || 0).toLocaleString()}`, color: 'indigo' },
    { title: '24h Revenue', value: `$${(stats?.revenue24h || 0).toLocaleString()}`, color: 'emerald' },
  ];

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-100'}`}>
      <div className="p-6">
        <h1 className={`text-3xl font-bold mb-6 ${isDark ? 'text-white' : 'text-gray-900'}`}>
          Dashboard
        </h1>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8">
          {statCards.map((stat, index) => (
            <div
              key={index}
              className={`p-6 rounded-lg shadow ${
                isDark ? 'bg-gray-800' : 'bg-white'
              }`}
            >
              <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                {stat.title}
              </p>
              <p className={`text-3xl font-bold mt-2 ${
                stat.color === 'blue' ? 'text-blue-600' :
                stat.color === 'green' ? 'text-green-600' :
                stat.color === 'yellow' ? 'text-yellow-600' :
                stat.color === 'purple' ? 'text-purple-600' :
                stat.color === 'indigo' ? 'text-indigo-600' :
                'text-emerald-600'
              }`}>
                {stat.value}
              </p>
            </div>
          ))}
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Users Chart */}
          <div className={`p-6 rounded-lg shadow ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              New Users (Last 7 Days)
            </h2>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#374151' : '#e5e7eb'} />
                <XAxis dataKey="name" stroke={isDark ? '#9ca3af' : '#6b7280'} />
                <YAxis stroke={isDark ? '#9ca3af' : '#6b7280'} />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: isDark ? '#1f2937' : '#fff',
                    border: 'none',
                    borderRadius: '8px'
                  }}
                />
                <Line 
                  type="monotone" 
                  dataKey="users" 
                  stroke="#3b82f6" 
                  strokeWidth={2}
                  dot={{ fill: '#3b82f6' }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Transactions Chart */}
          <div className={`p-6 rounded-lg shadow ${isDark ? 'bg-gray-800' : 'bg-white'}`}>
            <h2 className={`text-xl font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
              Transactions (Last 7 Days)
            </h2>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#374151' : '#e5e7eb'} />
                <XAxis dataKey="name" stroke={isDark ? '#9ca3af' : '#6b7280'} />
                <YAxis stroke={isDark ? '#9ca3af' : '#6b7280'} />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: isDark ? '#1f2937' : '#fff',
                    border: 'none',
                    borderRadius: '8px'
                  }}
                />
                <Bar dataKey="transactions" fill="#8b5cf6" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  );
}
