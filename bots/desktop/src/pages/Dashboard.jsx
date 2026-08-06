import React, { useState, useEffect } from 'react';

const API_URL = 'http://localhost:8105/api/v1';

function Dashboard() {
  const [balances, setBalances] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch(`${API_URL}/wallet/balances`)
      .then(res => res.json())
      .then(data => {
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
      ) : (
        <div className="balances-grid">
          {balances.map((balance, idx) => (
            <div key={idx} className="balance-card">
              <h3>{balance.token}</h3>
              <p className="network">{balance.network}</p>
              <p className="amount">{balance.balance}</p>
              <p className="value">${balance.usd_value}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default Dashboard;
