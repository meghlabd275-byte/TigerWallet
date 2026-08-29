// Launchpool Page — view pools, stake / unstake, list my stakes.
import React, { useState, useEffect, useCallback } from 'react';
import { api, WalletRecord } from '../services/api';

export default function Launchpool() {
  const [pools, setPools] = useState<unknown>(null);
  const [stakes, setStakes] = useState<unknown>(null);
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [amount, setAmount] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  const load = useCallback(() => {
    api.getLaunchpool().then(setPools).catch(() => setPools(null));
    api.getLaunchpoolStakes().then(setStakes).catch(() => setStakes(null));
  }, []);

  useEffect(() => {
    load();
    api.getWallets().then((d) => {
      setWallets(d.wallets || []);
      if (d.wallets && d.wallets.length > 0) setWalletId(d.wallets[0].id);
    }).catch(() => {});
  }, [load]);

  const act = async (action: 'stake' | 'unstake') => {
    setError(''); setResult(null);
    if (!walletId) { setError('Select a wallet'); return; }
    if (!amount) { setError('Enter an amount'); return; }
    setBusy(true);
    try {
      const params = { walletId, password, amount };
      const res = action === 'stake' ? await api.launchpoolStake(params) : await api.launchpoolUnstake(params);
      setResult(JSON.stringify(res));
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : `${action} failed`);
    } finally { setBusy(false); }
  };

  return (
    <div className="launchpool-page">
      <h1>Launchpool</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Action submitted to the blockchain network</h3><p className="mono">{result}</p></div>}
      <h2>Pools</h2>
      {pools ? <div className="quote-box"><pre>{JSON.stringify(pools, null, 2)}</pre></div> : <p className="empty-state">No launchpool data available.</p>}
      <div className="launchpool-form">
        <div className="form-group">
          <label>Wallet</label>
          <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
            {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)}</option>)}
          </select>
        </div>
        <div className="form-group"><label>Amount</label><input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} /></div>
        <div className="form-group"><label>Wallet Password</label><input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} /></div>
        <div className="action-row">
          <button className="primary-btn" onClick={() => act('stake')} disabled={busy}>Stake</button>
          <button className="secondary-btn" onClick={() => act('unstake')} disabled={busy}>Unstake</button>
        </div>
      </div>
      <h2>My Stakes</h2>
      {stakes ? <div className="quote-box"><pre>{JSON.stringify(stakes, null, 2)}</pre></div> : <p className="empty-state">No stakes yet.</p>}
    </div>
  );
}
