import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

function Dashboard() {
  const [balances, setBalances] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getBalances()
      .then((data) => {
        setBalances(data.balances || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  return (
    <div className="dashboard-page">
      <h1>Dashboard</h1>
      {loading ? (
        <p>Loading...</p>
      ) : balances.length === 0 ? (
        <p>No wallets found. Create one to get started.</p>
      ) : (
        <div className="balances-grid">
          {balances.map((b, idx) => (
            <div key={idx} className="balance-card">
              <h3>{b.symbol}</h3>
              <p className="network">Chain #{b.chain_id}</p>
              <p className="amount">{(b.balance_f || 0).toFixed(6)}</p>
              <p className="value">${(b.usd_value || 0).toFixed(2)}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default Dashboard;
