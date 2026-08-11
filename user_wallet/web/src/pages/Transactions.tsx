// Transactions Page
import React, { useState, useEffect } from 'react';
import { api, TransactionRecord } from '../services/api';

export default function Transactions() {
  const [transactions, setTransactions] = useState<TransactionRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState({ network: '', token: '' });

  useEffect(() => {
    loadTransactions();
  }, []);

  const loadTransactions = () => {
    api.getTransactions({ network: filter.network || undefined }).then((data) => {
      setTransactions(data.transactions || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  };

  return (
    <div className="transactions-page">
      <header className="page-header">
        <h1>Transactions</h1>
        <div className="filters">
          <select value={filter.network} onChange={(e) => setFilter({ ...filter, network: e.target.value })}>
            <option value="">All Networks</option>
            <option value="ethereum">Ethereum</option>
            <option value="bsc">BNB Chain</option>
            <option value="polygon">Polygon</option>
          </select>
          <select value={filter.token} onChange={(e) => setFilter({ ...filter, token: e.target.value })}>
            <option value="">All Tokens</option>
            <option value="ETH">ETH</option>
            <option value="USDT">USDT</option>
            <option value="USDC">USDC</option>
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
              <th>Tx Hash</th>
              <th>From</th>
              <th>To</th>
              <th>Value</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((tx) => (
              <tr key={tx.hash}>
                <td className="mono">{tx.hash.slice(0, 14)}...</td>
                <td className="mono">{tx.from.slice(0, 10)}...</td>
                <td className="mono">{tx.to.slice(0, 10)}...</td>
                <td>{tx.value}</td>
                <td><span className={`status ${tx.isError === '0' ? 'confirmed' : 'failed'}`}>{tx.isError === '0' ? 'Success' : 'Failed'}</span></td>
                <td>{new Date(parseInt(tx.timeStamp, 10) * 1000).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
