// Dashboard Page
import React from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { api, BalanceResult } from '../services/api';

export default function Dashboard() {
  const { user } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const [balances, setBalances] = React.useState<BalanceResult[]>([]);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    api.getBalances().then((data) => {
      setBalances(data.balances || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  return (
    <div className="dashboard">
      <header className="page-header">
        <h1>Welcome, {user?.username}</h1>
        <button onClick={toggleTheme} className="theme-toggle">
          {theme === 'light' ? '\uD83C\uDF19' : '\u2600\uFE0F'}
        </button>
      </header>

      <div className="stats-grid">
        <div className="stat-card">
          <h3>Total Balance (native)</h3>
          <p className="stat-value">
            {balances.reduce((sum: number, b: BalanceResult) => sum + (b.balance_f || 0), 0).toFixed(6)}
          </p>
        </div>
        <div className="stat-card">
          <h3>Wallets</h3>
          <p className="stat-value">{balances.length}</p>
        </div>
        <div className="stat-card">
          <h3>Networks</h3>
          <p className="stat-value">{new Set(balances.map((b: BalanceResult) => b.chain_id)).size}</p>
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
                  <th>Address</th>
                  <th>Balance</th>
                </tr>
              </thead>
              <tbody>
                {balances.map((b: BalanceResult, i: number) => (
                  <tr key={i}>
                    <td>{b.symbol}</td>
                    <td>Chain #{b.chain_id}</td>
                    <td className="mono">{b.address.slice(0, 10)}…{b.address.slice(-6)}</td>
                    <td>{b.balance_f.toFixed(6)}</td>
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
