/**
 * Admin Dashboard - Super Admin Panel
 * Connected to backend APIs
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface DashboardStats {
  totalUsers: number;
  totalVolume: number;
  activeWallets: number;
  totalTransactions: number;
  usersChange: number;
  volumeChange: number;
  walletsChange: number;
  transactionsChange: number;
}

interface RecentActivity {
  id: string;
  timestamp: string;
  action: string;
  admin: string;
  details: string;
}

interface WhiteLabelClient {
  id: string;
  name: string;
  status: string;
  authorized: boolean;
}

function Dashboard() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<DashboardStats>({
    totalUsers: 0,
    totalVolume: 0,
    activeWallets: 0,
    totalTransactions: 0,
    usersChange: 0,
    volumeChange: 0,
    walletsChange: 0,
    transactionsChange: 0,
  });
  const [activities, setActivities] = useState<RecentActivity[]>([]);
  const [clients, setClients] = useState<WhiteLabelClient[]>([]);
  const [error, setError] = useState<string | null>(null);

  const modules = [
    { name: 'User Management', path: '/users', icon: '👥', desc: 'Manage all platform users' },
    { name: 'White Label Clients', path: '/whitelabel', icon: '🏢', desc: 'White label client management' },
    { name: 'Token Management', path: '/tokens', icon: '🪙', desc: 'Manage tokens & cryptocurrencies' },
    { name: 'Pair Management', path: '/pairs', icon: '🔄', desc: 'Trading pair configuration' },
    { name: 'Liquidity Management', path: '/liquidity', icon: '💧', desc: 'Liquidity pool management' },
    { name: 'Fee Management', path: '/fees', icon: '💰', desc: 'Platform fee configuration' },
    { name: 'Blockchain Management', path: '/blockchains', icon: '⛓️', desc: 'Add/remove blockchain networks' },
    { name: 'KYC Management', path: '/kyc', icon: '🛡️', desc: 'User verification' },
    { name: 'Master Wallet', path: '/master-wallet', icon: '🔐', desc: 'Master wallet operations' },
    { name: 'Analytics', path: '/analytics', icon: '📊', desc: 'Platform analytics' },
    { name: 'API Management', path: '/api', icon: '🔌', desc: 'API keys & access' },
    { name: 'Settings', path: '/settings', icon: '⚙️', desc: 'Platform settings' },
  ];

  // Fetch dashboard data from backend
  const fetchDashboardData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      
      // Fetch dashboard stats
      const statsRes = await fetch(`${API_BASE_URL}/super-admin/dashboard`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (statsRes.ok) {
        const statsData = await statsRes.json();
        setStats({
          totalUsers: statsData.totalUsers || 0,
          totalVolume: statsData.totalVolume || 0,
          activeWallets: statsData.activeWallets || 0,
          totalTransactions: statsData.totalTransactions || 0,
          usersChange: statsData.usersChange || 0,
          volumeChange: statsData.volumeChange || 0,
          walletsChange: statsData.walletsChange || 0,
          transactionsChange: statsData.transactionsChange || 0,
        });
      }
      
      // Fetch recent activities
      const activityRes = await fetch(`${API_BASE_URL}/super-admin/logs?limit=10`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (activityRes.ok) {
        const activityData = await activityRes.json();
        setActivities(activityData.logs || []);
      }
      
      // Fetch white label clients for pending authorization count
      const clientsRes = await fetch(`${API_BASE_URL}/super-admin/clients`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (clientsRes.ok) {
        const clientsData = await clientsRes.json();
        setClients(clientsData.clients || []);
      }
      
    } catch (err) {
      console.error('Error fetching dashboard data:', err);
      setError('Failed to load dashboard data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  const pendingAuthCount = clients.filter(c => !c.authorized).length;
  const activeClientsCount = clients.filter(c => c.status === 'active').length;

  // Format number with K/M suffix
  const formatNumber = (num: number): string => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
  };

  // Format currency
  const formatCurrency = (num: number): string => {
    return '$' + formatNumber(num);
  };

  // Format percentage change
  const formatChange = (num: number): string => {
    const prefix = num >= 0 ? '+' : '';
    return prefix + num.toFixed(1) + '%';
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Super Admin Dashboard</h1>

      {error && (
        <div className="bg-red-500/20 border border-red-500 text-red-500 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Total Users</p>
          <p className="text-2xl font-bold">{formatNumber(stats.totalUsers)}</p>
          <p className="text-sm text-green-500">{formatChange(stats.usersChange)}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Total Volume</p>
          <p className="text-2xl font-bold">{formatCurrency(stats.totalVolume)}</p>
          <p className="text-sm text-green-500">{formatChange(stats.volumeChange)}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Active Wallets</p>
          <p className="text-2xl font-bold">{formatNumber(stats.activeWallets)}</p>
          <p className="text-sm text-green-500">{formatChange(stats.walletsChange)}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Transactions</p>
          <p className="text-2xl font-bold">{formatNumber(stats.totalTransactions)}</p>
          <p className="text-sm text-green-500">{formatChange(stats.transactionsChange)}</p>
        </div>
      </div>

      {/* White Label Status */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Active White Label Clients</p>
          <p className="text-2xl font-bold">{activeClientsCount}</p>
        </div>
        <div className="bg-slate-800 p-4 rounded-lg">
          <p className="text-sm opacity-60">Pending Authorizations</p>
          <p className="text-2xl font-bold text-amber-500">{pendingAuthCount}</p>
        </div>
      </div>

      {/* Modules Grid */}
      <h2 className="text-xl font-semibold mb-4">Management Modules</h2>
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {modules.map((mod, i) => (
          <div
            key={i}
            onClick={() => navigate(mod.path)}
            className="bg-slate-800 p-4 rounded-lg cursor-pointer hover:bg-slate-700 transition"
          >
            <div className="text-3xl mb-2">{mod.icon}</div>
            <h3 className="font-semibold">{mod.name}</h3>
            <p className="text-sm opacity-60">{mod.desc}</p>
          </div>
        ))}
      </div>

      {/* Recent Activity */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold mb-4">Recent Activity</h2>
        <div className="bg-slate-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-700">
              <tr>
                <th className="p-3 text-left">Time</th>
                <th className="p-3 text-left">Action</th>
                <th className="p-3 text-left">Admin</th>
                <th className="p-3 text-left">Details</th>
              </tr>
            </thead>
            <tbody>
              {activities.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-8 text-center opacity-60">No recent activity</td>
                </tr>
              ) : (
                activities.slice(0, 10).map((item, i) => (
                  <tr key={item.id || i} className="border-t border-slate-700">
                    <td className="p-3">{item.timestamp}</td>
                    <td className="p-3 text-amber-500">{item.action}</td>
                    <td className="p-3">{item.admin}</td>
                    <td className="p-3 opacity-60">{item.details}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
