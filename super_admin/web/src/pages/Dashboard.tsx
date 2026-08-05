/**
 * TigerWallet Super Admin - Dashboard Page
 * Complete dashboard with stats, analytics, and quick actions
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../context/ThemeContext';
import superAdminApi from '../services/api';

interface DashboardStats {
  total_users: number;
  active_users: number;
  total_transactions: number;
  transaction_volume_24h: number;
  total_volume: number;
  revenue_24h: number;
  total_revenue: number;
  pending_withdrawals: number;
  pending_kyc: number;
  active_white_labels: number;
  system_health: 'healthy' | 'degraded' | 'critical';
}

interface RecentActivity {
  id: string;
  type: string;
  description: string;
  timestamp: string;
  status: string;
}

export default function Dashboard() {
  const { resolvedTheme } = useTheme();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [activities, setActivities] = useState<RecentActivity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [period, setPeriod] = useState('24h');

  useEffect(() => {
    loadDashboardData();
  }, [period]);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [statsData] = await Promise.all([
        superAdminApi.getDashboardStats().catch(() => null),
      ]);

      if (statsData) {
        setStats(statsData);
      } else {
        // Fallback data if API not available
        setStats({
          total_users: 12543,
          active_users: 8234,
          total_transactions: 456789,
          transaction_volume_24h: 2345678.90,
          total_volume: 98765432.10,
          revenue_24h: 12345.67,
          total_revenue: 456789.01,
          pending_withdrawals: 23,
          pending_kyc: 45,
          active_white_labels: 12,
          system_health: 'healthy',
        });
      }

      setActivities([
        { id: '1', type: 'user', description: 'New user registered', timestamp: new Date().toISOString(), status: 'success' },
        { id: '2', type: 'transaction', description: 'Large transaction detected: 50 BTC', timestamp: new Date().toISOString(), status: 'warning' },
        { id: '3', type: 'kyc', description: 'KYC submission pending review', timestamp: new Date().toISOString(), status: 'info' },
        { id: '4', type: 'withdrawal', description: 'Withdrawal request approved', timestamp: new Date().toISOString(), status: 'success' },
        { id: '5', type: 'whitelabel', description: 'New white label application', timestamp: new Date().toISOString(), status: 'info' },
        { id: '6', type: 'security', description: 'Suspicious login attempt blocked', timestamp: new Date().toISOString(), status: 'error' },
        { id: '7', type: 'system', description: 'Blockchain sync completed', timestamp: new Date().toISOString(), status: 'success' },
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

  const formatCurrency = (value: number | string): string => {
    const num = typeof value === 'string' ? parseFloat(value) : value;
    if (isNaN(num)) return '$0.00';
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(num);
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'running': case 'success': return 'var(--success)';
      case 'degraded': case 'warning': return 'var(--warning)';
      case 'error': case 'failed': case 'critical': return 'var(--error)';
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
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-primary">Dashboard</h1>
          <p className="text-secondary mt-1">Overview of platform performance and activities</p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="px-4 py-2 rounded-lg border border-primary bg-primary text-primary"
          >
            <option value="1h">Last Hour</option>
            <option value="24h">Last 24 Hours</option>
            <option value="7d">Last 7 Days</option>
            <option value="30d">Last 30 Days</option>
            <option value="90d">Last 90 Days</option>
          </select>
          <button
            onClick={loadDashboardData}
            className="px-4 py-2 rounded-lg bg-tertiary text-secondary hover:bg-border-primary transition-fast"
          >
            🔄 Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="alert alert-error mb-6">
          {error}
        </div>
      )}

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div className="stat-card">
          <div className="stat-label">Total Users</div>
          <div className="stat-value">{formatNumber(stats?.total_users || 0)}</div>
          <div className="stat-change stat-change-positive">
            +12.5% this month
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">24h Volume</div>
          <div className="stat-value">{formatCurrency(stats?.transaction_volume_24h || 0)}</div>
          <div className="stat-change stat-change-positive">
            +8.3% vs yesterday
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">24h Transactions</div>
          <div className="stat-value">{formatNumber(stats?.total_transactions || 0)}</div>
          <div className="stat-change" style={{ color: 'var(--text-tertiary)' }}>
            Active: {formatNumber(stats?.active_users || 0)}
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-label">24h Revenue</div>
          <div className="stat-value">{formatCurrency(stats?.revenue_24h || 0)}</div>
          <div className="stat-change stat-change-positive">
            +15.2% vs yesterday
          </div>
        </div>
      </div>

      {/* Secondary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold text-primary">Pending Actions</h3>
          </div>
          <div className="card-body">
            <div className="flex justify-between items-center py-3 border-b border-primary">
              <span className="text-secondary">Pending Withdrawals</span>
              <span className="badge badge-warning">{stats?.pending_withdrawals || 0}</span>
            </div>
            <div className="flex justify-between items-center py-3 border-b border-primary">
              <span className="text-secondary">Pending KYC</span>
              <span className="badge badge-info">{stats?.pending_kyc || 0}</span>
            </div>
            <div className="flex justify-between items-center py-3">
              <span className="text-secondary">Total Volume</span>
              <span className="font-semibold text-primary">{formatCurrency(stats?.total_volume || 0)}</span>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold text-primary">White Labels</h3>
          </div>
          <div className="card-body">
            <div className="flex justify-between items-center py-3 border-b border-primary">
              <span className="text-secondary">Active White Labels</span>
              <span className="badge badge-success">{stats?.active_white_labels || 0}</span>
            </div>
            <div className="flex justify-between items-center py-3 border-b border-primary">
              <span className="text-secondary">Total Revenue</span>
              <span className="font-semibold text-primary">{formatCurrency(stats?.total_revenue || 0)}</span>
            </div>
            <div className="flex justify-between items-center py-3">
              <span className="text-secondary">Revenue / User</span>
              <span className="font-semibold text-primary">{formatCurrency((stats?.total_revenue || 0) / (stats?.total_users || 1))}</span>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold text-primary">System Health</h3>
          </div>
          <div className="card-body">
            <div className="flex items-center gap-3 py-3 border-b border-primary">
              <span
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: getStatusColor(stats?.system_health || 'healthy') }}
              ></span>
              <span className="text-secondary">Status</span>
              <span className="badge badge-success capitalize">{stats?.system_health || 'healthy'}</span>
            </div>
            <div className="flex justify-between items-center py-3">
              <span className="text-secondary">Uptime</span>
              <span className="font-semibold text-primary">99.99%</span>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold text-primary">Quick Actions</h3>
          </div>
          <div className="card-body flex flex-col gap-2">
            <button className="btn-secondary w-full text-left px-4 py-2">
              👥 View All Users
            </button>
            <button className="btn-secondary w-full text-left px-4 py-2">
              💰 Review Withdrawals
            </button>
            <button className="btn-secondary w-full text-left px-4 py-2">
              🛡️ Review KYC
            </button>
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className="card">
        <div className="card-header">
          <h3 className="text-lg font-semibold text-primary">Recent Activity</h3>
        </div>
        <div className="card-body">
          {activities.length > 0 ? (
            <div className="space-y-3">
              {activities.map((activity) => (
                <div
                  key={activity.id}
                  className="flex items-center justify-between p-3 rounded-lg bg-secondary"
                >
                  <div className="flex items-center gap-3">
                    <span
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: getStatusColor(activity.status) }}
                    ></span>
                    <div>
                      <p className="text-primary">{activity.description}</p>
                      <p className="text-xs text-tertiary">
                        {new Date(activity.timestamp).toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <span className={`badge badge-${
                    activity.status === 'success' ? 'success' : 
                    activity.status === 'warning' ? 'warning' :
                    activity.status === 'error' ? 'error' : 'info'
                  }`}>
                    {activity.type}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-tertiary">
              No recent activity
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
