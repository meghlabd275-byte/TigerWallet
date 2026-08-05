// TigerWallet Admin - Analytics Page
// Advanced analytics and reporting

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface AnalyticsData {
  totalUsers: number;
  activeUsers: number;
  totalTransactions: number;
  totalVolume: string;
  revenue: string;
  growth: string;
  userGrowth: number[];
  volumeHistory: number[];
  transactionHistory: number[];
}

const AnalyticsPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [period, setPeriod] = useState('30d');
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAnalytics();
  }, [period]);

  const loadAnalytics = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await adminApi.getAnalytics(period);
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load analytics');
    } finally {
      setLoading(false);
    }
  };

  const getColors = () => ({
    text: resolvedTheme === 'dark' ? '#f9fafb' : '#111827',
    textSecondary: resolvedTheme === 'dark' ? '#9ca3af' : '#6b7280',
    bgCard: resolvedTheme === 'dark' ? '#1e293b' : '#ffffff',
    border: resolvedTheme === 'dark' ? '#374151' : '#e5e7eb',
    primary: '#dc2626',
  });

  const colors = getColors();

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="loader"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold" style={{ color: colors.text }}>Analytics</h1>
        <select
          className="form-select"
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          style={{ backgroundColor: colors.bgCard, color: colors.text, borderColor: colors.border }}
        >
          <option value="7d">Last 7 days</option>
          <option value="30d">Last 30 days</option>
          <option value="90d">Last 90 days</option>
          <option value="1y">Last year</option>
        </select>
      </div>

      {error && <div className="alert alert-error mb-4">{error}</div>}

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div className="stat-card" style={{ backgroundColor: colors.bgCard }}>
          <div className="stat-label" style={{ color: colors.textSecondary }}>Total Users</div>
          <div className="stat-value" style={{ color: colors.text }}>{data?.totalUsers || 0}</div>
        </div>
        <div className="stat-card" style={{ backgroundColor: colors.bgCard }}>
          <div className="stat-label" style={{ color: colors.textSecondary }}>Active Users</div>
          <div className="stat-value" style={{ color: colors.text }}>{data?.activeUsers || 0}</div>
        </div>
        <div className="stat-card" style={{ backgroundColor: colors.bgCard }}>
          <div className="stat-label" style={{ color: colors.textSecondary }}>Transactions</div>
          <div className="stat-value" style={{ color: colors.text }}>{data?.totalTransactions || 0}</div>
        </div>
        <div className="stat-card" style={{ backgroundColor: colors.bgCard }}>
          <div className="stat-label" style={{ color: colors.textSecondary }}>Volume</div>
          <div className="stat-value" style={{ color: colors.text }}>${data?.totalVolume || '0'}</div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card" style={{ backgroundColor: colors.bgCard }}>
          <div className="card-header">
            <h3 className="text-lg font-semibold" style={{ color: colors.text }}>User Growth</h3>
          </div>
          <div className="card-body">
            <div className="h-48 flex items-end gap-2">
              {(data?.userGrowth || [10, 20, 30, 40, 50, 60, 70]).map((value, index) => (
                <div key={index} className="flex-1 bg-red-500 rounded-t" style={{ height: `${value}%` }}></div>
              ))}
            </div>
          </div>
        </div>

        <div className="card" style={{ backgroundColor: colors.bgCard }}>
          <div className="card-header">
            <h3 className="text-lg font-semibold" style={{ color: colors.text }}>Transaction Volume</h3>
          </div>
          <div className="card-body">
            <div className="h-48 flex items-end gap-2">
              {(data?.transactionHistory || [15, 25, 35, 45, 55, 65, 75]).map((value, index) => (
                <div key={index} className="flex-1 bg-blue-500 rounded-t" style={{ height: `${value}%` }}></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AnalyticsPage;
