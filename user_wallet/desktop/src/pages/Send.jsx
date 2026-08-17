import React, { useState, useEffect } from 'react';
import { api, parsePaymentUri } from '../services/api';

const CHAIN_OPTIONS = [
  { value: 'ethereum', label: 'Ethereum', id: 1 },
  { value: 'bsc', label: 'BNB Chain', id: 56 },
  { value: 'polygon', label: 'Polygon', id: 137 },
];
const CHAIN_IDS = { ethereum: 1, bsc: 56, polygon: 137 };

function Send() {
  const [wallets, setWallets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({
    walletId: '',
    network: 'ethereum',
    to: '',
    value: '',
    password: '',
  });
  const [qrInput, setQrInput] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(null);

  useEffect(() => {
    api.getWallets()
      .then((data) => {
        const list = data.wallets || [];
        setWallets(list);
        if (list.length > 0 && !form.walletId) {
          setForm((f) => ({ ...f, walletId: list[0].id || list[0].wallet_id || '' }));
        }
        setLoading(false);
      })
      .catch(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const applyQr = () => {
    const parsed = parsePaymentUri(qrInput);
    if (!parsed) {
      setError('Could not parse a valid address from QR input');
      return;
    }
    setError('');
    setForm((f) => ({
      ...f,
      to: parsed.address || f.to,
      value: parsed.amount !== undefined ? parsed.amount : f.value,
      network: parsed.chainId && parsed.chainId === 56 ? 'bsc'
        : parsed.chainId === 137 ? 'polygon'
        : 'ethereum',
    }));
    setQrInput('');
  };

  const buildPayload = () => ({
    walletId: form.walletId,
    password: form.password,
    to: form.to.trim(),
    value: form.value,
    chainId: CHAIN_IDS[form.network] || 1,
  });

  const doSend = async (auto) => {
    setError('');
    setSuccess(null);
    if (!form.walletId) { setError('Select a wallet'); return; }
    if (!form.to.trim()) { setError('Recipient address is required'); return; }
    if (form.password.length < 8) { setError('Password must be at least 8 characters'); return; }
    setBusy(true);
    try {
      const res = auto
        ? await api.autoSendTransaction(buildPayload())
        : await api.sendTransaction(buildPayload());
      const hash = res && (res.hash || res.tx_hash || res.transactionHash || res.txHash);
      setSuccess({ auto, hash });
    } catch (err) {
      setError(err.message || 'Transaction failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="send-page">
      <h1>Send</h1>

      {success && (
        <div className="success-banner">
          <h3>✓ Transaction submitted to the blockchain network</h3>
          {success.hash && <p className="mono">Tx hash: {success.hash}</p>}
          {success.auto && <p className="mono">Auto-approved</p>}
          <button className="link-btn" onClick={() => setSuccess(null)}>Dismiss</button>
        </div>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : wallets.length === 0 ? (
        <p>No wallets yet. Create one first to send funds.</p>
      ) : (
        <form className="send-form" onSubmit={(e) => { e.preventDefault(); doSend(false); }}>
          {error && <div className="error">{error}</div>}
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

          <label>Recipient address</label>
          <input
            placeholder="0x..."
            value={form.to}
            onChange={(e) => setForm({ ...form, to: e.target.value })}
            required
          />

          <label>Amount</label>
          <input
            type="text"
            inputMode="decimal"
            placeholder="0.0"
            value={form.value}
            onChange={(e) => setForm({ ...form, value: e.target.value })}
            required
          />

          <label>Wallet password</label>
          <input
            type="password"
            placeholder="Password (min 8 chars)"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            required
            minLength={8}
          />

          <label>Paste QR / payment URI</label>
          <div className="qr-row">
            <input
              placeholder="ethereum:0x...&value=0.1"
              value={qrInput}
              onChange={(e) => setQrInput(e.target.value)}
            />
            <button type="button" onClick={applyQr}>Apply</button>
          </div>

          <div className="send-actions">
            <button type="submit" disabled={busy}>{busy ? 'Sending…' : 'Send Transaction'}</button>
            <button type="button" disabled={busy} onClick={() => doSend(true)}>
              ⚡ Auto-Send (auto-approved)
            </button>
          </div>
        </form>
      )}
    </div>
  );
}

export default Send;
