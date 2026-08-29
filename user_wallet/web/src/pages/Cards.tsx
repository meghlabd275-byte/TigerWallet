// Crypto Card Page — card balance, live rates, and card transactions.
import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';

export default function Cards() {
  const [rates, setRates] = useState<unknown>(null);
  const [balance, setBalance] = useState<unknown>(null);
  const [txs, setTxs] = useState<unknown[]>([]);
  const [error, setError] = useState('');

  const load = useCallback(() => {
    api.getCryptoCardRates().then(setRates).catch(() => setRates(null));
    api.getCryptoCardBalance().then(setBalance).catch(() => setBalance(null));
    api.getCardTransactions().then((d) => {
      const list = Array.isArray(d) ? d : ((d as Record<string, unknown>).transactions as unknown[] || []);
      setTxs(list);
    }).catch(() => setTxs([]));
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="cards-page">
      <h1>Crypto Card</h1>
      {error && <div className="error">{error}</div>}
      <h2>Balance</h2>
      {balance ? <div className="quote-box"><pre>{JSON.stringify(balance, null, 2)}</pre></div> : <p className="empty-state">No card balance available.</p>}
      <h2>Rates</h2>
      {rates ? <div className="quote-box"><pre>{JSON.stringify(rates, null, 2)}</pre></div> : <p className="empty-state">Rates unavailable.</p>}
      <h2>Card Transactions</h2>
      {txs.length === 0 && <p className="empty-state">No card transactions.</p>}
      <ul className="record-list">
        {txs.map((t, i) => (
          <li key={i} className="record-item"><span className="mono">{JSON.stringify(t)}</span></li>
        ))}
      </ul>
    </div>
  );
}
