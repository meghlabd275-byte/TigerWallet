import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function Staking() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [quote, setQuote] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [wallets, setWallets] = useState([]);
  const [form, setForm] = useState({
    walletId: '',
    network: 'ethereum',
    asset: '',
    amount: '',
    password: '',
  });
  const [busy, setBusy] = useState(false);
  const [action, setAction] = useState('stake');
  const [info, setInfo] = useState('');

  useEffect(() => {
    let alive = true;
    setLoading(true);
    Promise.all([api.getStakingQuote(), api.getWallets()])
      .then(([q, w]) => {
        if (!alive) return;
        setQuote(q);
        const list = w.wallets || [];
        setWallets(list);
        if (list.length > 0) {
          setForm((f) => ({ ...f, walletId: list[0].id || list[0].wallet_id || '' }));
        }
        setLoading(false);
      })
      .catch((err) => { if (alive) { setError(err.message || 'Failed to load staking quote'); setLoading(false); } });
    return () => { alive = false; };
  }, []);

  const assets = (quote && (quote.assets || quote.supported_assets)) || [];

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setInfo('');
    if (!form.walletId) { setError('Select a wallet'); return; }
    if (!form.asset) { setError('Select a staking asset'); return; }
    if (action !== 'claim' && !form.amount) { setError('Amount is required'); return; }
    if (form.password.length < 8) { setError('Wallet password is required (min 8 chars)'); return; }
    setBusy(true);
    try {
      const params = {
        walletId: form.walletId,
        password: form.password,
        asset: form.asset,
        chainId: CHAIN_IDS[form.network] || 1,
      };
      if (action !== 'claim') params.amount = form.amount;
      let res;
      if (action === 'stake') res = await api.stake(params);
      else if (action === 'unstake') res = await api.unstake(params);
      else res = await api.claim(params);
      const hash = res && (res.hash || res.tx_hash || res.transactionHash || res.txHash);
      setInfo(hash
        ? `${action === 'stake' ? 'Stake' : action === 'unstake' ? 'Unstake' : 'Claim'} submitted — Tx hash: ${hash}`
        : `${action} submitted to the blockchain network.`);
    } catch (err) {
      setError(err.message || `${action} failed`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="send-page">
      <h1>Staking</h1>

      {loading ? (
        <p>Loading...</p>
      ) : error && !quote ? (
        <div className="error">{error}</div>
      ) : (
        <>
          <div className="success-banner" style={{ marginBottom: '20px' }}>
            <h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>Supported staking assets</h3>
            {quote && (quote.apy !== undefined || quote.min_stake !== undefined || quote.lock_period !== undefined) && (
              <p className="mono" style={{ marginTop: '6px' }}>
                {quote.apy !== undefined && <>APY: {quote.apy} </>}
                {quote.min_stake !== undefined && <>· Min stake: {quote.min_stake} </>}
                {quote.lock_period !== undefined && <>· Lock: {quote.lock_period}</>}
              </p>
            )}
            {assets.length === 0 ? (
              <p>No supported assets returned.</p>
            ) : (
              <div className="wallets-grid" style={{ marginTop: '12px' }}>
                {assets.map((a, idx) => (
                  <div key={idx} className="wallet-card">
                    <h3>{a.symbol || a.asset || a.name || a}</h3>
                    {a.apy !== undefined && <p className="network">APY: {a.apy}</p>}
                    {a.min_stake !== undefined && <p className="network">Min: {a.min_stake}</p>}
                  </div>
                ))}
              </div>
            )}
          </div>

          {wallets.length === 0 ? (
            <p>No wallets yet. Create one first to stake.</p>
          ) : (
            <form className="send-form" onSubmit={handleSubmit}>
              {error && <div className="error">{error}</div>}
              {info && <div className="success-banner"><h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>✓ {info}</h3></div>}

              <label>Action</label>
              <select value={action} onChange={(e) => setAction(e.target.value)}>
                <option value="stake">Stake</option>
                <option value="unstake">Unstake</option>
                <option value="claim">Claim rewards</option>
              </select>

              <label>Wallet</label>
              <select value={form.walletId} onChange={(e) => setForm({ ...form, walletId: e.target.value })}>
                {wallets.map((w, idx) => (
                  <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                    {w.label} — {w.address ? w.address.slice(0, 10) : ''}…
                  </option>
                ))}
              </select>

              <label>Chain</label>
              <select value={form.network} onChange={(e) => setForm({ ...form, network: e.target.value })}>
                {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
              </select>

              <label>Asset</label>
              <input
                placeholder="e.g. ETH"
                value={form.asset}
                onChange={(e) => setForm({ ...form, asset: e.target.value })}
                required
              />

              {action !== 'claim' && (
                <>
                  <label>Amount</label>
                  <input
                    type="text"
                    inputMode="decimal"
                    placeholder="0.0"
                    value={form.amount}
                    onChange={(e) => setForm({ ...form, amount: e.target.value })}
                    required
                  />
                </>
              )}

              <label>Wallet password</label>
              <input
                type="password"
                placeholder="Password (min 8 chars)"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                minLength={8}
              />

              <div className="send-actions">
                <button type="submit" disabled={busy}>
                  {busy ? 'Submitting…' : action === 'stake' ? 'Stake' : action === 'unstake' ? 'Unstake' : 'Claim'}
                </button>
              </div>
            </form>
          )}
        </>
      )}
    </div>
  );
}

export default Staking;
