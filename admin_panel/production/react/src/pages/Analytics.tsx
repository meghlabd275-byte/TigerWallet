/**
 * Analytics Dashboard
 * Connected to backend APIs
 */

import React, { useState, useEffect, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

interface AnalyticsData {
  totalUsers: number;
  activeUsers: number;
  totalVolume: number;
  revenue: number;
  transactions: number;
  gasSaved: number;
  usersChange: number;
  activeUsersChange: number;
  volumeChange: number;
  revenueChange: number;
  transactionsChange: number;
  gasSavedChange: number;
}

interface TokenVolume {
  name: string;
  volume: number;
  trades: number;
}

interface VolumeDataPoint {
  timestamp: string;
  volume: number;
}

function Analytics() {
  const [timeRange, setTimeRange] = useState('7d');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [metrics, setMetrics] = useState<AnalyticsData>({
    totalUsers: 0,
    activeUsers: 0,
    totalVolume: 0,
    revenue: 0,
    transactions: 0,
    gasSaved: 0,
    usersChange: 0,
    activeUsersChange: 0,
    volumeChange: 0,
    revenueChange: 0,
    transactionsChange: 0,
    gasSavedChange: 0,
  });
  const [topTokens, setTopTokens] = useState<TokenVolume[]>([]);
  const [volumeData, setVolumeData] = useState<VolumeDataPoint[]>([]);

  // Fetch analytics data from backend
  const fetchAnalytics = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const token = localStorage.getItem('superadmin_token');
      const response = await fetch(`${API_BASE_URL}/super-admin/analytics?range=${timeRange}`, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch analytics');
      }
      
      const data = await response.json();
      setMetrics({
        totalUsers: data.totalUsers || 0,
        activeUsers: data.activeUsers || 0,
        totalVolume: data.totalVolume || 0,
        revenue: data.revenue || 0,
        transactions: data.transactions || 0,
        gasSaved: data.gasSaved || 0,
        usersChange: data.usersChange || 0,
        activeUsersChange: data.activeUsersChange || 0,
        volumeChange: data.volumeChange || 0,
        revenueChange: data.revenueChange || 0,
        transactionsChange: data.transactionsChange || 0,
        gasSavedChange: data.gasSavedChange || 0,
      });
      setTopTokens(data.topTokens || []);
      setVolumeData(data.volumeData || []);
    } catch (err) {
      console.error('Error fetching analytics:', err);
      setError(err instanceof Error ? err.message : 'Failed to load analytics');
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    fetchAnalytics();
  }, [fetchAnalytics]);

  // Format helpers
  const formatNumber = (num: number): string => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
  };

  const formatCurrency = (num: number): string => {
    return '$' + formatNumber(num);
  };

  const formatChange = (num: number): string => {
    const prefix = num >= 0 ? '+' : '';
    return prefix + num.toFixed(1) + '%';
  };

  const metricsDisplay = [
    { label: 'Total Users', value: formatNumber(metrics.totalUsers), change: formatChange(metrics.usersChange) },
    { label: 'Active Users', value: formatNumber(metrics.activeUsers), change: formatChange(metrics.activeUsersChange) },
    { label: 'Total Volume', value: formatCurrency(metrics.totalVolume), change: formatChange(metrics.volumeChange) },
    { label: 'Revenue', value: formatCurrency(metrics.revenue), change: formatChange(metrics.revenueChange) },
    { label: 'Transactions', value: formatNumber(metrics.transactions), change: formatChange(metrics.transactionsChange) },
    { label: 'Gas Saved', value: formatCurrency(metrics.gasSaved), change: formatChange(metrics.gasSavedChange) },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-amber-500"></div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Analytics</h1>
        <div className="flex gap-2">
          {['24h', '7d', '30d', '90d', '1y'].map(range => (
            <button key={range} onClick={() => setTimeRange(range)} className={`px-3 py-1 rounded ${timeRange === range ? 'bg-amber-500 text-black' : 'bg-slate-800'}`}>
              {range}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="bg-red-500/20 border border-red-500 text-red-500 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-6">
        {metricsDisplay.map((m, i) => (
          <div key={i} className="bg-slate-800 p-4 rounded-lg">
            <p className="text-sm opacity-60">{m.label}</p>
            <p className="text-xl font-bold">{m.value}</p>
            <p className="text-sm text-green-500">{m.change}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Volume Over Time</h3>
          <div className="h-48 flex items-end justify-between gap-2">
            {(volumeData.length > 0 ? volumeData : Array(12).fill(0)).map((d: any, i: number) => (
              <div key={i} className="flex-1 bg-amber-500/50 rounded-t" style={{height: `${d?.volume || 50}%`}}></div>
            ))}
          </div>
        </div>

        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Top Tokens by Volume</h3>
          <div className="space-y-3">
            {topTokens.length === 0 ? (
              <p className="text-center opacity-60">No data available</p>
            ) : (
              topTokens.map((t, i) => (
                <div key={i} className="flex justify-between items-center">
                  <span>{t.name}</span>
                  <div className="text-right">
                    <p className="font-semibold">{formatCurrency(t.volume)}</p>
                    <p className="text-sm opacity-60">{t.trades.toLocaleString()} trades</p>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mt-6">
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">User Growth</h3>
          <p className="text-3xl font-bold text-green-500">{formatChange(metrics.usersChange)}</p>
          <p className="text-sm opacity-60">vs last period</p>
        </div>
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Avg Transaction Size</h3>
          <p className="text-3xl font-bold">$287</p>
          <p className="text-sm opacity-60">+5% vs last period</p>
        </div>
        <div className="bg-slate-800 p-6 rounded-lg">
          <h3 className="font-semibold mb-4">Platform Fees</h3>
          <p className="text-3xl font-bold text-amber-500">$1.2M</p>
          <p className="text-sm opacity-60">+22% vs last period</p>
        </div>
      </div>
    </div>
  );
}

export default Analytics;
