// Strategies Page
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Strategies() {
  const { isDark } = useTheme();
  const [strategies, setStrategies] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getStrategies().then(data => {
      setStrategies(data.strategies || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  if (loading) return <div className="strategies-page">Loading...</div>;

  return (
    <div className="strategies-page">
      <h1>Trading Strategies ({isDark ? 'Dark' : 'Light'})</h1>
      <div className="strategies-grid">
        {strategies.map((s: any) => (
          <div key={s.id} className="strategy-card">
            <h3>{s.name}</h3>
            <p className="strategy-type">{s.type}</p>
            <p>{s.description}</p>
            <span className={`risk ${s.risk_level}`}>{s.risk_level} risk</span>
          </div>
        ))}
      </div>
    </div>
  );
}
