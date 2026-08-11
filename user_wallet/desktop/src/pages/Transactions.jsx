import React, { useState, useEffect } from 'react';
import { api } from '../services/api';

function Transactions() {
  const [transactions, setTransactions] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getTransactions()
      .then((data) => {
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
              <th>Tx Hash</th>
              <th>From</th>
              <th>To</th>
              <th>Value</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((tx, idx) => (
              <tr key={idx}>
                <td className="mono">{tx.hash.slice(0, 14)}...</td>
                <td className="mono">{tx.from.slice(0, 10)}...</td>
                <td className="mono">{tx.to.slice(0, 10)}...</td>
                <td>{tx.value}</td>
                <td>
                  <span className={`status ${tx.isError === '0' ? 'confirmed' : 'failed'}`}>
                    {tx.isError === '0' ? 'Success' : 'Failed'}
                  </span>
                </td>
                <td>{new Date(parseInt(tx.timeStamp, 10) * 1000).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

export default Transactions;
