// Dashboard Page - Bots
import React, { useState, useEffect } from 'react';
import { api, Bot } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Dashboard() {
  const { isDark } = useTheme();
  const [stats, setStats] = useState({ totalBots: 0, running: 0, stopped: 0, paused: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getBots().then(data => {
      const bots: Bot[] = data.bots || [];
      setStats({
        totalBots: bots.length,
        running: bots.filter(b => b.status === 'running').length,
        stopped: bots.filter(b => b.status === 'stopped').length,
        paused: bots.filter(b => b.status === 'paused').length,
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
          <h3>Running</h3>
          <p className="stat-value">{stats.running}</p>
        </div>
        <div className="stat-card">
          <h3>Paused</h3>
          <p className="stat-value">{stats.paused}</p>
        </div>
        <div className="stat-card">
          <h3>Stopped</h3>
          <p className="stat-value">{stats.stopped}</p>
        </div>
      </div>
    </div>
  );
}
