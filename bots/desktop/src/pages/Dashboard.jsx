import React, { useEffect, useState } from 'react';
import api from '../services/bot-api';
import { useTheme } from '../contexts/ThemeContext';

export default function Dashboard() {
  const { isDark } = useTheme();
  const [stats, setStats] = useState({ totalBots: 0, running: 0, stopped: 0, paused: 0 });
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    api.listBots()
      .then((data) => {
        if (!alive) return;
        const bots = (data && data.bots) || [];
        setStats({
          totalBots: bots.length,
          running: bots.filter(b => b.status === 'running').length,
          stopped: bots.filter(b => b.status === 'stopped').length,
          paused: bots.filter(b => b.status === 'paused').length,
        });
      })
      .catch((e) => { if (alive) setError(e.message); })
      .finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  if (loading) return <div className="dashboard">Loading…</div>;

  return (
    <div className="dashboard">
      <h1>Bots Dashboard ({isDark ? 'Dark' : 'Light'} mode)</h1>
      {error && <p className="error-banner">{error}</p>}
      <div className="stats-grid">
        <div className="stat-card"><h3>Total Bots</h3><p className="stat-value">{stats.totalBots}</p></div>
        <div className="stat-card"><h3>Running</h3><p className="stat-value">{stats.running}</p></div>
        <div className="stat-card"><h3>Paused</h3><p className="stat-value">{stats.paused}</p></div>
        <div className="stat-card"><h3>Stopped</h3><p className="stat-value">{stats.stopped}</p></div>
      </div>
    </div>
  );
}
