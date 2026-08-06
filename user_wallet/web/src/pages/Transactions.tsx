// Transactions Page
import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

export default function Transactions() {
  const [transactions, setTransactions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState({ network: '', token: '' });

  useEffect(() => {
    loadTransactions();
  }, []);

  const loadTransactions = () => {
    api.getTransactions(filter).then(data => {
      setTransactions(data.transactions || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  };

  return (
    <div className="transactions-page">
      <header className="page-header">
        <h1>Transactions</h1>
        <div className="filters">
          <select value={filter.network} onChange={e => setFilter({...filter, network: e.target.value})}>
            <option value="">All Networks</option>
            <option value="ethereum">Ethereum</option>
            <option value="bsc">BNB Chain</option>
            <option value="polygon">Polygon</option>
          </select>
          <select value={filter.token} onChange={e => setFilter({...filter, token: e.target.value})}>
            <option value="">All Tokens</option>
            <option value="ETH">ETH</option>
            <option value="BTC">BTC</option>
            <option value="USDT">USDT</option>
          </select>
          <button onClick={loadTransactions}>Apply</button>
        </div>
      </header>

      {loading ? <p>Loading...</p> : transactions.length === 0 ? (
        <p>No transactions found.</p>
      ) : (
        <table className="transactions-table">
          <thead>
            <tr>
              <th>Type</th>
              <th>Token</th>
              <th>Amount</th>
              <th>Network</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((tx: any) => (
              <tr key={tx.id}>
                <td>{tx.type}</td>
                <td>{tx.token}</td>
                <td>{tx.amount}</td>
                <td>{tx.network}</td>
                <td><span className={`status ${tx.status}`}>{tx.status}</span></td>
                <td>{new Date(tx.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
