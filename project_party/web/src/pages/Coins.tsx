// Coins Page - ProjectParty
import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

export default function Coins() {
  const [coins, setCoins] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [network, setNetwork] = useState('');

  useEffect(() => { loadCoins(); }, [network]);

  const loadCoins = () => {
    api.getCoins(network || undefined).then(data => {
      setCoins(data.coins || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div className="coins-page">
      <h1>All Coins</h1>
      <select value={network} onChange={e => setNetwork(e.target.value)}>
        <option value="">All Networks</option>
        <option value="bitcoin">Bitcoin</option>
        <option value="ethereum">Ethereum</option>
        <option value="bsc">BNB Chain</option>
      </select>

      <div className="coins-table">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>Name</th>
              <th>Symbol</th>
              <th>Price</th>
              <th>24h %</th>
              <th>Market Cap</th>
              <th>Volume</th>
            </tr>
          </thead>
          <tbody>
            {coins.map((c: any) => (
              <tr key={c.id}>
                <td>{c.rank}</td>
                <td><img src={c.icon_url} alt={c.name} /> {c.name}</td>
                <td>{c.symbol}</td>
                <td>${c.price}</td>
                <td className={c.change_24h >= 0 ? 'up' : 'down'}>{c.change_24h}%</td>
                <td>${c.market_cap}</td>
                <td>${c.volume_24h}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
