// Staking Page — view supported staking assets, stake / unstake / claim.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';

export default function Staking() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [asset, setAsset] = useState('ETH');
  const [amount, setAmount] = useState('');
  const [quote, setQuote] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<string | null>(null);

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) setWalletId(data.wallets[0].id);
    }).catch(() => {});
    api.getStakingQuote().then(setQuote).catch(() => {});
  }, []);

  const act = async (action: 'stake' | 'unstake' | 'claim') => {
    setError('');
    setResult(null);
    if (!walletId) { setError('Select a wallet'); return; }
    setBusy(true);
    try {
      let res: unknown;
      if (action === 'stake') res = await api.stake({ walletId, password, asset, amount, chainId: 1 });
      else if (action === 'unstake') res = await api.unstake({ walletId, password, asset, amount, chainId: 1 });
      else res = await api.claim({ walletId, password, asset, chainId: 1 });
      setResult(typeof res === 'string' ? res : JSON.stringify(res));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : `${action} failed`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="staking-page">
      <h1>Staking</h1>
      {error && <div className="error">{error}</div>}
      {result && <div className="success-banner"><h3>✓ Action submitted to the blockchain network</h3><p className="mono">{result}</p></div>}
      {Boolean(quote) && <div className="quote-box"><pre>{JSON.stringify(quote, null, 2)}</pre></div>}
      <div className="staking-form">
        <div className="form-group">
          <label>Wallet</label>
          <select value={walletId} onChange={(e) => setWalletId(e.target.value)}>
            {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)}</option>)}
          </select>
        </div>
        <div className="form-group">
          <label>Asset</label>
          <input value={asset} onChange={(e) => setAsset(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Amount</label>
          <input type="number" step="any" value={amount} onChange={(e) => setAmount(e.target.value)} />
        </div>
        <div className="form-group">
          <label>Wallet Password</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} />
        </div>
        <div className="action-row">
          <button className="primary-btn" onClick={() => act('stake')} disabled={busy}>Stake</button>
          <button className="secondary-btn" onClick={() => act('unstake')} disabled={busy}>Unstake</button>
          <button className="secondary-btn" onClick={() => act('claim')} disabled={busy}>Claim</button>
        </div>
      </div>
    </div>
  );
}
