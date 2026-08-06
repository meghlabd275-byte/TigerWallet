import React, { useState, useEffect } from 'react';

const API_URL = 'http://localhost:8105/api/v1';

function Transactions() {
  const [transactions, setTransactions] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch(`${API_URL}/wallet/transactions`)
      .then(res => res.json())
      .then(data => {
        setTransactions(data.transactions || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  return (
    <div className="transactions-page">
      <h1>Transactions</h1>
      {loading ? (
        <p>Loading...</p>
      ) : transactions.length === 0 ? (
        <p>No transactions yet.</p>
      ) : (
        <table className="transactions-table">
          <thead>
            <tr>
              <th>Type</th>
              <th>Token</th>
              <th>Amount</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((tx, idx) => (
              <tr key={idx}>
                <td>{tx.type}</td>
                <td>{tx.token}</td>
                <td>{tx.amount}</td>
                <td><span className={`status ${tx.status.toLowerCase()}`}>{tx.status}</span></td>
                <td>{new Date(tx.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

export default Transactions;
