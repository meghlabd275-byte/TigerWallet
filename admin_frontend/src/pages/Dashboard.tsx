/**
 * TigerWallet Admin Dashboard
 * Complete dashboard with statistics and overview
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import api from '../services/api';

interface DashboardStats {
  active_white_labels: number;
  pending_white_labels: number;
  total_users: number;
  active_users: number;
  transactions_24h: number;
  total_revenue: number;
  total_admins: number;
  total_products: number;
}

const Dashboard: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      setLoading(true);
      const response = await api.getDashboardStats();
      if (response.data) {
        setStats(response.data.stats);
      } else {
        setError(response.error || 'Failed to load stats');
      }
    } catch (err) {
      setError('Failed to connect to server');
    } finally {
      setLoading(false);
    }
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(amount);
  };

  const formatNumber = (num: number) => {
    return new Intl.NumberFormat('en-US').format(num);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--bg-primary)] flex items-center justify-center">
        <div className="text-[var(--text-primary)] text-xl">Loading...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[var(--bg-primary)]">
      {/* Header */}
      <header className="bg-[var(--bg-secondary)] border-b border-[var(--border-color)] px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-[var(--text-primary)]">TigerWallet Admin</h1>
            <p className="text-[var(--text-muted)]">Platform Overview</p>
          </div>
          <button
            onClick={toggleTheme}
            className="p-2 rounded-lg bg-[var(--bg-tertiary)] hover:bg-[var(--hover-bg)] transition-colors"
          >
            {theme === 'dark' ? (
              <svg className="w-6 h-6 text-[var(--text-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            ) : (
              <svg className="w-6 h-6 text-[var(--text-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
              </svg>
            )}
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="p-6">
        {error && (
          <div className="mb-6 p-4 bg-[var(--error)] bg-opacity-10 border border-[var(--error)] rounded-lg text-[var(--error)]">
            {error}
            <button onClick={loadStats} className="ml-4 underline">Retry</button>
          </div>
        )}

        {/* Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {/* Active White Labels */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-[var(--text-secondary)] text-sm font-medium">Active White Labels</h3>
              <span className="w-3 h-3 bg-[var(--success)] rounded-full"></span>
            </div>
            <p className="text-3xl font-bold text-[var(--text-primary)]">
              {stats ? formatNumber(stats.active_white_labels) : '0'}
            </p>
            <p className="text-[var(--text-muted)] text-sm mt-2">
              {stats?.pending_white_labels || 0} pending
            </p>
          </div>

          {/* Total Users */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-[var(--text-secondary)] text-sm font-medium">Total Users</h3>
              <span className="w-3 h-3 bg-[var(--accent-primary)] rounded-full"></span>
            </div>
            <p className="text-3xl font-bold text-[var(--text-primary)]">
              {stats ? formatNumber(stats.total_users) : '0'}
            </p>
            <p className="text-[var(--text-muted)] text-sm mt-2">
              {stats ? formatNumber(stats.active_users) : '0'} active
            </p>
          </div>

          {/* Transactions (24h) */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-[var(--text-secondary)] text-sm font-medium">Transactions (24h)</h3>
              <span className="w-3 h-3 bg-[var(--accent-secondary)] rounded-full"></span>
            </div>
            <p className="text-3xl font-bold text-[var(--text-primary)]">
              {stats ? formatNumber(stats.transactions_24h) : '0'}
            </p>
            <p className="text-[var(--text-muted)] text-sm mt-2">
              Last 24 hours
            </p>
          </div>

          {/* Total Revenue */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-[var(--text-secondary)] text-sm font-medium">Total Revenue</h3>
              <span className="w-3 h-3 bg-[var(--warning)] rounded-full"></span>
            </div>
            <p className="text-3xl font-bold text-[var(--text-primary)]">
              {stats ? formatCurrency(stats.total_revenue) : '$0.00'}
            </p>
            <p className="text-[var(--text-muted)] text-sm mt-2">
              All time
            </p>
          </div>
        </div>

        {/* Secondary Stats */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {/* Total Admins */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <h3 className="text-[var(--text-secondary)] text-sm font-medium mb-2">Total Admins</h3>
            <p className="text-2xl font-bold text-[var(--text-primary)]">
              {stats ? formatNumber(stats.total_admins) : '0'}
            </p>
          </div>

          {/* Total Products */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <h3 className="text-[var(--text-secondary)] text-sm font-medium mb-2">Total Products</h3>
            <p className="text-2xl font-bold text-[var(--text-primary)]">
              {stats ? formatNumber(stats.total_products) : '0'}
            </p>
          </div>

          {/* System Status */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <h3 className="text-[var(--text-secondary)] text-sm font-medium mb-2">System Status</h3>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 bg-[var(--success)] rounded-full animate-pulse"></span>
              <span className="text-[var(--text-primary)] font-medium">Operational</span>
            </div>
          </div>

          {/* Last Updated */}
          <div className="bg-[var(--card-bg)] rounded-xl p-6 border border-[var(--border-color)] shadow-sm">
            <h3 className="text-[var(--text-secondary)] text-sm font-medium mb-2">Last Updated</h3>
            <p className="text-[var(--text-primary)] font-medium">
              {new Date().toLocaleTimeString()}
            </p>
            <button 
              onClick={loadStats}
              className="text-[var(--accent-primary)] text-sm mt-2 hover:underline"
            >
              Refresh
            </button>
          </div>
        </div>
      </main>
    </div>
  );
};

export default Dashboard;
