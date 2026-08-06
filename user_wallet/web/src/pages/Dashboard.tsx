// Dashboard Page
import React from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { useTheme } from '../../contexts/ThemeContext';
import { api } from '../../services/api';

export default function Dashboard() {
  const { user } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const [balances, setBalances] = React.useState<any[]>([]);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    api.getBalances().then(data => {
      setBalances(data.balances || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  return (
    <div className="dashboard">
      <header className="page-header">
        <h1>Welcome, {user?.username}</h1>
        <button onClick={toggleTheme} className="theme-toggle">
          {theme === 'light' ? '🌙' : '☀️'}
        </button>
      </header>

      <div className="stats-grid">
        <div className="stat-card">
          <h3>Total Balance</h3>
          <p className="stat-value">
            ${balances.reduce((sum: number, b: any) => sum + parseFloat(b.balance || '0'), 0).toFixed(2)}
          </p>
        </div>
        <div className="stat-card">
          <h3>Wallets</h3>
          <p className="stat-value">{balances.length}</p>
        </div>
        <div className="stat-card">
          <h3>Networks</h3>
          <p className="stat-value">{new Set(balances.map((b: any) => b.network)).size}</p>
        </div>
      </div>

      {loading ? <p>Loading...</p> : (
        <div className="balances-list">
          <h2>Your Balances</h2>
          {balances.length === 0 ? (
            <p>No wallets found. Create one to get started.</p>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Token</th>
                  <th>Network</th>
                  <th>Balance</th>
                </tr>
              </thead>
              <tbody>
                {balances.map((b: any, i: number) => (
                  <tr key={i}>
                    <td>{b.token}</td>
                    <td>{b.network}</td>
                    <td>{b.balance}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}
