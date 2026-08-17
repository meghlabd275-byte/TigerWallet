import React, { useState, useEffect } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { api } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function Swap() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);

  const [form, setForm] = useState({
    walletId: '',
    network: 'ethereum',
    fromToken: '',
    toToken: '',
    amount: '',
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
    if (!form.fromToken.trim() || !form.toToken.trim() || !form.amount) {
      setError('From token, to token and amount are required for a quote');
      return;
    }
    setQuoteBusy(true);
    try {
      const res = await api.getSwapQuote({
        fromToken: form.fromToken.trim(),
        toToken: form.toToken.trim(),
        fromAmount: form.amount,
        chainId: CHAIN_IDS[form.network] || 1,
      });
      setQuote(res);
    } catch (err) {
      setError(err.message || 'Failed to fetch swap quote');
    } finally {
      setQuoteBusy(false);
    }
  };

  const handleSwap = async (e) => {
    e.preventDefault();
    setError('');
    setSuccess(null);
    if (!form.walletId) { setError('Select a signing wallet'); return; }
    if (!form.fromToken.trim() || !form.toToken.trim() || !form.amount) {
      setError('From token, to token and amount are required'); return;
    }
    if (form.password.length < 8) { setError('Wallet password is required (min 8 chars)'); return; }
    setBusy(true);
    try {
      const res = await api.ammSwap({
        walletId: form.walletId,
        password: form.password,
        fromToken: form.fromToken.trim(),
        toToken: form.toToken.trim(),
        fromAmount: form.amount,
        chainId: CHAIN_IDS[form.network] || 1,
      });
      const hash = res && (res.hash || res.tx_hash || res.transactionHash || res.txHash);
      setSuccess({ hash });
    } catch (err) {
      setError(err.message || 'Swap failed');
    } finally {
      setBusy(false);
    }
  };

  const quoteOut = quote && (quote.to_amount || quote.toAmount || quote.out_amount || quote.outAmount);

  return (
    <div className="send-page">
      <h1>Swap</h1>

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
        <p>No wallets yet. Create one first to swap.</p>
      ) : (
        <form className="send-form" onSubmit={handleSwap}>
          {error && <div className="error">{error}</div>}

          <label>Signing wallet</label>
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

          <label>From token</label>
          <input
            placeholder="e.g. ETH or 0x…"
            value={form.fromToken}
            onChange={(e) => setForm({ ...form, fromToken: e.target.value })}
            required
          />

          <label>To token</label>
          <input
            placeholder="e.g. USDC or 0x…"
            value={form.toToken}
            onChange={(e) => setForm({ ...form, toToken: e.target.value })}
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
              {quoteBusy ? 'Quoting…' : 'Get Quote'}
            </button>
            <button type="submit" disabled={busy}>{busy ? 'Swapping…' : 'Swap'}</button>
          </div>

          {quote && (
            <div className="success-banner" style={{ marginTop: '16px' }}>
              <h3 style={{ color: isDark ? '#4CAF50' : 'var(--accent)' }}>Indicative quote</h3>
              <p className="mono">
                {form.amount} {form.fromToken} ≈ {quoteOut !== undefined ? quoteOut : '—'} {form.toToken}
              </p>
              {quote.price && <p className="mono">Price: {quote.price}</p>}
              {quote.minimum_received !== undefined && (
                <p className="mono">Min received: {quote.minimum_received}</p>
              )}
            </div>
          )}
        </form>
      )}
    </div>
  );
}

export default Swap;
