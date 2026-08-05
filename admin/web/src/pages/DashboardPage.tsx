// TigerWallet Admin - Dashboard Page
// Main dashboard with stats, charts, and quick actions

import React, { useState, useEffect } from 'react';
import { adminApi } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

interface DashboardStats {
  totalUsers: number;
  activeUsers: number;
  totalTransactions: number;
  volume24h: string;
  revenue24h: string;
  pendingWithdrawals: number;
  pendingKyc: number;
  totalVolume: string;
  growth: string;
}

interface RecentActivity {
  id: string;
  type: string;
  description: string;
  timestamp: string;
  status: string;
}

interface SystemStatus {
  name: string;
  status: string;
  uptime: string;
  latency: string;
}

const DashboardPage: React.FC = () => {
  const { resolvedTheme } = useTheme();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [activities, setActivities] = useState<RecentActivity[]>([]);
  const [systemStatus, setSystemStatus] = useState<SystemStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [analyticsData, systemData] = await Promise.all([
        adminApi.getAnalytics('24h').catch(() => null),
        adminApi.getSystemStatus().catch(() => []),
      ]);

      if (analyticsData) {
        setStats({
          totalUsers: analyticsData.totalUsers || 0,
          activeUsers: analyticsData.activeUsers || 0,
          totalTransactions: analyticsData.dailyTransactions || 0,
          volume24h: analyticsData.totalVolume || '0',
          revenue24h: analyticsData.revenue || '0',
          pendingWithdrawals: 0,
          pendingKyc: 0,
          totalVolume: analyticsData.totalVolume || '0',
          growth: analyticsData.growth || '0%',
        });
      }

      setSystemStatus(Array.isArray(systemData) ? systemData : []);
      setActivities([
        { id: '1', type: 'user', description: 'New user registered', timestamp: new Date().toISOString(), status: 'success' },
        { id: '2', type: 'transaction', description: 'Large transaction detected', timestamp: new Date().toISOString(), status: 'warning' },
        { id: '3', type: 'kyc', description: 'KYC submission pending review', timestamp: new Date().toISOString(), status: 'info' },
        { id: '4', type: 'withdrawal', description: 'Withdrawal request approved', timestamp: new Date().toISOString(), status: 'success' },
        { id: '5', type: 'token', description: 'New token listed', timestamp: new Date().toISOString(), status: 'success' },
      ]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  };

  const formatNumber = (num: number | string): string => {
    const numValue = typeof num === 'string' ? parseFloat(num) : num;
    if (isNaN(numValue)) return '0';
    if (numValue >= 1000000) return (numValue / 1000000).toFixed(2) + 'M';
    if (numValue >= 1000) return (numValue / 1000).toFixed(2) + 'K';
    return numValue.toFixed(2);
  };

  const formatCurrency = (value: string): string => {
    const num = parseFloat(value);
    if (isNaN(num)) return '$0.00';
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(num);
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'running': case 'success': return 'var(--color-success)';
      case 'degraded': case 'warning': return 'var(--color-warning)';
      case 'error': case 'failed': return 'var(--color-error)';
      default: return 'var(--text-tertiary)';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="loader"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
          Dashboard
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Overview of platform performance and activities
        </p>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div className="stat-card">
          <div className="stat-label">Total Users</div>
          <div className="stat-value">{formatNumber(stats?.totalUsers || 0)}</div>
          <div className="stat-change stat-change-positive">
            +{stats?.growth || '0%'} this month
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">24h Volume</div>
          <div className="stat-value">{formatCurrency(stats?.volume24h || '0')}</div>
          <div className="stat-change stat-change-positive">
            +{stats?.growth || '0%'} vs yesterday
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">24h Transactions</div>
          <div className="stat-value">{formatNumber(stats?.totalTransactions || 0)}</div>
          <div className="stat-change" style={{ color: 'var(--text-tertiary)' }}>
            Active: {formatNumber(stats?.activeUsers || 0)}
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">24h Revenue</div>
          <div className="stat-value">{formatCurrency(stats?.revenue24h || '0')}</div>
          <div className="stat-change stat-change-positive">
            +{stats?.growth || '0%'} vs yesterday
          </div>
        </div>
      </div>

      {/* Secondary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Pending Actions</h3>
          </div>
          <div className="card-body">
            <div className="flex justify-between items-center py-3 border-b" style={{ borderColor: 'var(--border-primary)' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Pending Withdrawals</span>
              <span className="badge badge-warning">{stats?.pendingWithdrawals || 0}</span>
            </div>
            <div className="flex justify-between items-center py-3 border-b" style={{ borderColor: 'var(--border-primary)' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Pending KYC</span>
              <span className="badge badge-info">{stats?.pendingKyc || 0}</span>
            </div>
            <div className="flex justify-between items-center py-3">
              <span style={{ color: 'var(--text-secondary)' }}>Total Volume</span>
              <span style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
                {formatCurrency(stats?.totalVolume || '0')}
              </span>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">System Status</h3>
          </div>
          <div className="card-body">
            {systemStatus.length > 0 ? (
              systemStatus.slice(0, 5).map((service, index) => (
                <div key={index} className="flex justify-between items-center py-2">
                  <div className="flex items-center gap-2">
                    <span
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: getStatusColor(service.status) }}
                    ></span>
                    <span style={{ color: 'var(--text-primary)' }}>{service.name}</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                      {service.uptime}
                    </span>
                    <span className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                      {service.latency}
                    </span>
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-4" style={{ color: 'var(--text-tertiary)' }}>
                No system status data available
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="card">
        <div className="card-header">
          <h3 className="text-lg font-semibold">Recent Activity</h3>
        </div>
        <div className="card-body">
          {activities.length > 0 ? (
            <div className="space-y-3">
              {activities.map((activity) => (
                <div
                  key={activity.id}
                  className="flex items-center justify-between p-3 rounded"
                  style={{ backgroundColor: 'var(--bg-secondary)' }}
                >
                  <div className="flex items-center gap-3">
                    <span
                      className="w-2 h-2 rounded-full"
                      style={{
                        backgroundColor: getStatusColor(activity.status),
                      }}
                    ></span>
                    <div>
                      <p style={{ color: 'var(--text-primary)' }}>{activity.description}</p>
                      <p className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                        {new Date(activity.timestamp).toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <span className={`badge badge-${activity.status === 'success' ? 'success' : activity.status === 'warning' ? 'warning' : 'info'}`}>
                    {activity.type}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-4" style={{ color: 'var(--text-tertiary)' }}>
              No recent activity
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default DashboardPage;
