// Trades Page
import React, { useState, useEffect } from 'react';
import { api } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Trades() {
  const { isDark } = useTheme();
  const [trades, setTrades] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getTrades().then(data => {
      setTrades(data.trades || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  if (loading) return <div className="trades-page">Loading...</div>;

  return (
    <div className="trades-page">
      <h1>Trade History ({isDark ? 'Dark' : 'Light'})</h1>
      {trades.length === 0 ? (
        <p>No trades yet.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Pair</th>
              <th>Type</th>
              <th>Amount</th>
              <th>Price</th>
              <th>Profit</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {trades.map((t: any) => (
              <tr key={t.id}>
                <td>{t.pair}</td>
                <td>{t.order_type}</td>
                <td>{t.amount}</td>
                <td>{t.price}</td>
                <td className={parseFloat(t.profit || '0') >= 0 ? 'profit' : 'loss'}>{t.profit}</td>
                <td>{t.status}</td>
                <td>{new Date(t.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
