// Dashboard Page - Main Admin Overview
// Complete statistics and management overview

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import './Dashboard.css';

interface Stats {
  totalUsers: number;
  activeWallets: number;
  totalTransactions: number;
  dailyVolume: number;
  totalRevenue: number;
  activeChains: number;
}

interface RecentTransaction {
  id: string;
  user: string;
  type: string;
  amount: string;
  status: string;
  time: string;
}

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const [stats, setStats] = useState<Stats>({
    totalUsers: 0,
    activeWallets: 0,
    totalTransactions: 0,
    dailyVolume: 0,
    totalRevenue: 0,
    activeChains: 0,
  });
  const [recentTransactions, setRecentTransactions] = useState<RecentTransaction[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Load dashboard data
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    setIsLoading(true);
    try {
      // Simulated API call - replace with actual API
      // const response = await fetch('/api/admin/dashboard');
      // const data = await response.json();
      
      // Mock data
      setStats({
        totalUsers: 125847,
        activeWallets: 98234,
        totalTransactions: 3847291,
        dailyVolume: 284756390,
        totalRevenue: 8472930,
        activeChains: 87,
      });

      setRecentTransactions([
        { id: '1', user: '0x7a23...8f91', type: 'Transfer', amount: '5.5 ETH', status: 'completed', time: '2 min ago' },
        { id: '2', user: '0x3b14...2c78', type: 'Swap', amount: '12,450 USDT', status: 'completed', time: '5 min ago' },
        { id: '3', user: 'Sol123...456', type: 'Stake', amount: '250 SOL', status: 'pending', time: '8 min ago' },
        { id: '4', user: '0x9f42...1a63', type: 'Bridge', amount: '1,000 USDC', status: 'completed', time: '12 min ago' },
        { id: '5', user: '0x2e87...9b12', type: 'Mint NFT', amount: '1 BAYC', status: 'completed', time: '15 min ago' },
      ]);
    } catch (error) {
      console.error('Failed to load dashboard:', error);
    }
    setIsLoading(false);
  };

  const formatNumber = (num: number): string => {
    if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(2) + 'K';
    return num.toString();
  };

  const formatCurrency = (num: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(num);
  };

  if (isLoading) {
    return (
      <div className="dashboard-loading">
        <div className="spinner"></div>
        <p>Loading dashboard...</p>
      </div>
    );
  }

  return (
    <div className="dashboard">
      <div className="page-header">
        <h1>Dashboard</h1>
        <p>Welcome back, Super Admin</p>
      </div>

      {/* Stats Cards */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon">👥</div>
          <div className="stat-content">
            <span className="stat-label">Total Users</span>
            <span className="stat-value">{formatNumber(stats.totalUsers)}</span>
            <span className="stat-change positive">+12.5% this month</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">💼</div>
          <div className="stat-content">
            <span className="stat-label">Active Wallets</span>
            <span className="stat-value">{formatNumber(stats.activeWallets)}</span>
            <span className="stat-change positive">+8.3% this month</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">📝</div>
          <div className="stat-content">
            <span className="stat-label">Total Transactions</span>
            <span className="stat-value">{formatNumber(stats.totalTransactions)}</span>
            <span className="stat-change positive">+24.7% this month</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-content">
            <span className="stat-label">Daily Volume</span>
            <span className="stat-value">{formatCurrency(stats.dailyVolume)}</span>
            <span className="stat-change positive">+15.2% today</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-content">
            <span className="stat-label">Total Revenue</span>
            <span className="stat-value">{formatCurrency(stats.totalRevenue)}</span>
            <span className="stat-change positive">+18.9% this month</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">⛓️</div>
          <div className="stat-content">
            <span className="stat-label">Active Chains</span>
            <span className="stat-value">{stats.activeChains}</span>
            <span className="stat-change">Active now</span>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="quick-actions">
        <h2>Quick Actions</h2>
        <div className="actions-grid">
          <button className="action-card" onClick={() => navigate('/users')}>
            <span className="action-icon">👥</span>
            <span className="action-label">Manage Users</span>
          </button>
          <button className="action-card" onClick={() => navigate('/blockchain')}>
            <span className="action-icon">⛓️</span>
            <span className="action-label">Add Blockchain</span>
          </button>
          <button className="action-card" onClick={() => navigate('/pairs')}>
            <span className="action-icon">🔄</span>
            <span className="action-label">Manage Pairs</span>
          </button>
          <button className="action-card" onClick={() => navigate('/whitelabel')}>
            <span className="action-icon">🏢</span>
            <span className="action-label">White Label</span>
          </button>
          <button className="action-card" onClick={() => navigate('/fees')}>
            <span className="action-icon">💰</span>
            <span className="action-label">Fee Settings</span>
          </button>
          <button className="action-card" onClick={() => navigate('/analytics')}>
            <span className="action-icon">📈</span>
            <span className="action-label">Analytics</span>
          </button>
        </div>
      </div>

      {/* Recent Transactions */}
      <div className="recent-transactions">
        <div className="section-header">
          <h2>Recent Transactions</h2>
          <button className="view-all-btn" onClick={() => navigate('/transactions')}>
            View All
          </button>
        </div>
        
        <div className="transactions-table">
          <table>
            <thead>
              <tr>
                <th>User</th>
                <th>Type</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {recentTransactions.map(tx => (
                <tr key={tx.id}>
                  <td className="user-cell">
                    <span className="user-address">{tx.user}</span>
                  </td>
                  <td>
                    <span className={`type-badge type-${tx.type.toLowerCase()}`}>
                      {tx.type}
                    </span>
                  </td>
                  <td className="amount-cell">{tx.amount}</td>
                  <td>
                    <span className={`status-badge status-${tx.status}`}>
                      {tx.status}
                    </span>
                  </td>
                  <td className="time-cell">{tx.time}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* System Status */}
      <div className="system-status">
        <h2>System Status</h2>
        <div className="status-grid">
          <div className="status-item">
            <span className="status-indicator online"></span>
            <span className="status-label">API Gateway</span>
            <span className="status-value">Online</span>
          </div>
          <div className="status-item">
            <span className="status-indicator online"></span>
            <span className="status-label">Database</span>
            <span className="status-value">Online</span>
          </div>
          <div className="status-item">
            <span className="status-indicator online"></span>
            <span className="status-label">RPC Nodes</span>
            <span className="status-value">87/87 Active</span>
          </div>
          <div className="status-item">
            <span className="status-indicator online"></span>
            <span className="status-label">Background Workers</span>
            <span className="status-value">Online</span>
          </div>
          <div className="status-item">
            <span className="status-indicator online"></span>
            <span className="status-label">Cache Server</span>
            <span className="status-value">Online</span>
          </div>
          <div className="status-item">
            <span className="status-indicator warning"></span>
            <span className="status-label">Queue</span>
            <span className="status-value">Processing</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
