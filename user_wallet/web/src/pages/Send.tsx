// Send Page — real on-chain transfer via /send (or /auto-send for auto-approval).
// Shows "Transaction submitted to the blockchain network" on success.
import React, { useState, useEffect } from 'react';
import { api, WalletRecord } from '../services/api';
import { parsePaymentUri } from '../services/api';

const CHAIN_OPTIONS = [
  { id: 1, label: 'Ethereum' },
  { id: 56, label: 'BNB Chain' },
  { id: 137, label: 'Polygon' },
  { id: 42161, label: 'Arbitrum' },
  { id: 10, label: 'Optimism' },
  { id: 8453, label: 'Base' },
  { id: 43114, label: 'Avalanche' },
];

export default function Send() {
  const [wallets, setWallets] = useState<WalletRecord[]>([]);
  const [walletId, setWalletId] = useState('');
  const [password, setPassword] = useState('');
  const [to, setTo] = useState('');
  const [amount, setAmount] = useState('');
  const [chainId, setChainId] = useState(1);
  const [qrInput, setQrInput] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<{ hash: string; autoApproved: boolean } | null>(null);

  useEffect(() => {
    api.getWallets().then((data) => {
      setWallets(data.wallets || []);
      if (data.wallets && data.wallets.length > 0) {
        setWalletId(data.wallets[0].id);
        setChainId(data.wallets[0].chain_id);
      }
    }).catch(() => {});
  }, []);

  const applyQr = () => {
    const parsed = parsePaymentUri(qrInput);
    if (parsed) {
      setTo(parsed.address);
      if (parsed.amount) setAmount(parsed.amount);
      if (parsed.chainId) setChainId(parsed.chainId);
      setQrInput('');
    } else {
      setError('Could not parse QR / payment URI');
    }
  };

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setResult(null);
    if (!walletId) { setError('Select a wallet'); return; }
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) { setError('Invalid recipient address'); return; }
    if (!amount || parseFloat(amount) <= 0) { setError('Enter a valid amount'); return; }
    setBusy(true);
    try {
      const res = await api.sendTransaction({ walletId, password, to, value: amount, chainId });
      setResult({ hash: res.transaction_hash, autoApproved: false });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Send failed');
    } finally {
      setBusy(false);
    }
  };

  const handleAutoSend = async (e: React.MouseEvent) => {
    e.preventDefault();
    setError('');
    setResult(null);
    if (!walletId) { setError('Select a wallet'); return; }
    if (!/^0x[a-fA-F0-9]{40}$/.test(to)) { setError('Invalid recipient address'); return; }
    if (!amount || parseFloat(amount) <= 0) { setError('Enter a valid amount'); return; }
    setBusy(true);
    try {
      const res = await api.autoSendTransaction({ walletId, password, to, value: amount, chainId });
      setResult({ hash: res.transaction_hash, autoApproved: res.auto_approved });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Auto-send failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="send-page">
      <h1>Send</h1>
      {error && <div className="error">{error}</div>}
      {result && (
        <div className="success-banner">
          <h3>✓ Transaction submitted to the blockchain network</h3>
          <p>Your transaction has been broadcast and is awaiting confirmation.</p>
          {result.autoApproved && <p className="auto-badge">⚡ Auto-approved by MasterWallet</p>}
          <p className="mono tx-hash">{result.hash}</p>
        </div>
      )}
      <form onSubmit={handleSend} className="send-form">
        <div className="form-group">
          <label>From Wallet</label>
          <select value={walletId} onChange={(e) => setWalletId(e.target.value)} required>
            {wallets.map((w) => <option key={w.id} value={w.id}>{w.label || w.address.slice(0, 10)} · {w.address.slice(0, 8)}…</option>)}
          </select>
        </div>
        <div className="form-group">
          <label>Network</label>
          <select value={chainId} onChange={(e) => setChainId(Number(e.target.value))}>
            {CHAIN_OPTIONS.map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
          </select>
        </div>
        <div className="form-group">
          <label>Recipient Address</label>
          <input placeholder="0x…" value={to} onChange={(e) => setTo(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Amount</label>
          <input type="number" step="any" placeholder="0.0" value={amount} onChange={(e) => setAmount(e.target.value)} required />
        </div>
        <div className="form-group">
          <label>Wallet Password</label>
          <input type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={8} />
        </div>
        <div className="qr-paste">
          <label>Scan / paste payment URI</label>
          <div className="qr-row">
            <input placeholder="ethereum:0x…?value=1" value={qrInput} onChange={(e) => setQrInput(e.target.value)} />
            <button type="button" onClick={applyQr}>Apply</button>
          </div>
        </div>
        <button type="submit" className="primary-btn" disabled={busy}>{busy ? 'Sending…' : 'Send'}</button>
        <button type="button" className="secondary-btn" onClick={handleAutoSend} disabled={busy}>⚡ Auto-Send (auto-approved)</button>
      </form>
    </div>
  );
}
