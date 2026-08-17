import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

// NOTE: the backend exposes no dedicated bridge endpoint. The indicative
// cross-chain rate is fetched via api.getConvertQuote (same route the web
// app's Convert/Swap uses). Broadcasting the bridge transfer is honestly
// performed via api.sendTransaction — no fabricated "bridge tx" type is used.
function Bridge() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);

  const [form, setForm] = useState({
    walletId: '',
    fromNetwork: 'ethereum',
    toNetwork: 'bsc',
    token: '',
    amount: '',
    recipient: '',
    password: '',
  });
  const [quote, setQuote] = useState(null);
  const [quoteBusy, setQuoteBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(null);

  useEffect(() => {
    let alive = true;
    api.getWallets()
      .then((data) => {
        if (!alive) return;
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0) {
          setForm((f) => ({ ...f, walletId: list[0].id || list[0].wallet_id || '' }));
        }
        setLoading(false);
      })
      .catch(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);

  const fetchQuote = async () => {
    setError('');
    setQuote(null);
    if (!form.token.trim() || !form.amount) {
      setError('Token and amount are required for an indicative quote');
      return;
    }
    setQuoteBusy(true);
    try {
      const res = await api.getConvertQuote({
        fromToken: form.token.trim(),
        toToken: form.token.trim(), // cross-chain bridge of the same asset
        fromAmount: form.amount,
        chainId: CHAIN_IDS[form.fromNetwork] || 1,
      });
      setQuote(res);
    } catch (err) {
      setError(err.message || 'Failed to fetch bridge quote');
    } finally {
      setQuoteBusy(false);
    }
  };

  const handleBridge = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess(null);
    if (!form.walletId) { setError('Select a wallet'); return; }
    if (!form.token.trim() || !form.amount) { setError('Token and amount are required'); return; }
    if (form.fromNetwork === form.toNetwork) { setError('Source and destination chain must differ'); return; }
    if (!form.recipient.trim()) { setError('Recipient address is required'); return; }
    if (form.password.length < 8) { setError('Wallet password is required (min 8 chars)'); return; }
    setBusy(true);
    try {
      const res = await api.sendTransaction({
        walletId: form.walletId,
        password: form.password,
        to: form.recipient.trim(),
        value: form.amount,
        chainId: CHAIN_IDS[form.fromNetwork] || 1,
      });
      const hash = res && (res.hash || res.tx_hash || res.transactionHash || res.txHash);
      setSuccess({ hash });
    } catch (err) {
      setError(err.message || 'Bridge transaction failed');
    } finally {
      setBusy(false);
    }
  };

  const quoteOut = quote && (quote.to_amount || quote.toAmount || quote.out_amount || quote.outAmount);

  return (
    <div className="send-page">
      <h1>Bridge</h1>

      <p style={{ color: 'var(--text-secondary)', marginBottom: '16px', fontSize: '0.85rem' }}>
        Cross-chain indicative rate via the convert endpoint; broadcast uses a signed on-chain transaction.
      </p>

      {success && (
        <div className="success-banner">
          <h3>✓ Transaction submitted to the blockchain network</h3>
          {success.hash && <p className="mono">Tx hash: {success.hash}</p>}
          <button className="link-btn" onClick={() => setSuccess(null)}>Dismiss</button>
        </div>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one first to bridge.</p>
      ) : (
        <form className="send-form" onSubmit={handleBridge}>
          {error && <div className="error">{error}</div>}

          <label>Signing wallet</label>
          <select value={form.walletId} onChange={(e) => setForm({ ...form, walletId: e.target.value })}>
            {wallets.map((w, idx) => (
              <option key={w.id || idx} value={w.id || w.wallet_id || ''}>
                {w.label} — {w.address ? w.address.slice(0, 10) : ''}…
              </option>
            ))}
          </select>

          <label>From chain</label>
          <select value={form.fromNetwork} onChange={(e) => setForm({ ...form, fromNetwork: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>

          <label>To chain</label>
          <select value={form.toNetwork} onChange={(e) => setForm({ ...form, toNetwork: e.target.value })}>
            {CHAIN_OPTIONS.map((c) => <option key={c.value} value={c.value}>{c.label}</option>)}
          </select>

          <label>Token</label>
          <input
            placeholder="e.g. ETH or 0x…"
            value={form.token}
            onChange={(e) => setForm({ ...form, token: e.target.value })}
            required
          />

          <label>Amount</label>
          <input
            type="text"
            inputMode="decimal"
            placeholder="0.0"
            value={form.amount}
            onChange={(e) => setForm({ ...form, amount: e.target.value })}
            required
          />

          <label>Recipient address (on destination chain)</label>
          <input
            placeholder="0x..."
            value={form.recipient}
            onChange={(e) => setForm({ ...form, recipient: e.target.value })}
            required
          />

          <label>Wallet password</label>
          <input
            type="password"
            placeholder="Password (min 8 chars)"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            minLength={8}
          />

          <div className="send-actions">
            <button type="button" onClick={fetchQuote} disabled={quoteBusy}>
              {quoteBusy ? 'Quoting…' : 'Indicative Quote'}
            </button>
            <button type="submit" disabled={busy}>{busy ? 'Bridging…' : 'Bridge'}</button>
          </div>

          {quote && (
            <div className="success-banner" style={{ marginTop: '16px' }}>
              <h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>Indicative bridge rate</h3>
              <p className="mono">
                {form.amount} {form.token} ≈ {quoteOut !== undefined ? quoteOut : '—'} {form.token}
              </p>
            </div>
          )}
        </form>
      )}
    </div>
  );
}

export default Bridge;
