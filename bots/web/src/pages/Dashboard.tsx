// Dashboard Page - Bots
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Dashboard() {
  const { isDark } = useTheme();
  const [stats, setStats] = useState({ totalBots: 0, activeBots: 0, totalProfit: '$0', totalTrades: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getBots().then(data => {
      const bots = data.bots || [];
      setStats({
        totalBots: bots.length,
        activeBots: bots.filter((b: any) => b.status === 'running').length,
        totalProfit: bots.reduce((sum: number, b: any) => sum + parseFloat(b.total_profit || '0'), 0).toFixed(2),
        totalTrades: bots.reduce((sum: number, b: any) => sum + (b.total_trades || 0), 0)
      });
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  if (loading) return <div className="dashboard">Loading...</div>;

  return (
    <div className="dashboard">
      <h1>Trading Bots Dashboard ({isDark ? 'Dark' : 'Light'} mode)</h1>
      <div className="stats-grid">
        <div className="stat-card">
          <h3>Total Bots</h3>
          <p className="stat-value">{stats.totalBots}</p>
        </div>
        <div className="stat-card">
          <h3>Active Bots</h3>
          <p className="stat-value">{stats.activeBots}</p>
        </div>
        <div className="stat-card">
          <h3>Total Profit</h3>
          <p className="stat-value">${stats.totalProfit}</p>
        </div>
        <div className="stat-card">
          <h3>Total Trades</h3>
          <p className="stat-value">{stats.totalTrades}</p>
        </div>
      </div>
    </div>
  );
}
